package bao

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/openbao/openbao/api"
)

const (
	HAClusterType string = "HA"
)

const (
	ClusterJsonName = "cluster.json"
)

type Cluster struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Nodes []string `json:"nodes"`
}

func BuildHACluster(clusterName string, nodeName string) (*Cluster, error) {
	c := &Cluster{
		Name: clusterName,
		Type: HAClusterType,
	}

	node, err := LoadNode(nodeName)
	if err != nil {
		return nil, fmt.Errorf("error loading node to add to cluster: %w", err)
	}

	if node.Cluster != "" {
		return nil, fmt.Errorf("node `%v` already present in different cluster: `%v`", nodeName, node.Cluster)
	}

	if node.Config.Dev != nil {
		return nil, fmt.Errorf("node `%v` is a dev-node cluster; ephemeral nodes cannot be added to clusters", nodeName)
	}

	node.Cluster = clusterName
	if err := node.SaveConfig(); err != nil {
		return nil, fmt.Errorf("failed saving node `%v` while adding to cluster: %w", nodeName, err)
	}

	c.Nodes = append(c.Nodes, nodeName)

	if err := c.SaveConfig(); err != nil {
		return nil, fmt.Errorf("error saving cluster configuration: %w", err)
	}

	return c, nil
}

func ListClusters() ([]string, error) {
	dir := ClusterBaseDirectory()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cluster directory (%v): %w", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error listing cluster directory (`%v`): %w", dir, err)
	}

	var results []string
	for _, entry := range entries {
		if entry.IsDir() {
			results = append(results, entry.Name())
		}
	}

	return results, nil
}

func ClusterExists(name string) (bool, error) {
	clusters, err := ListClusters()
	if err != nil {
		return false, err
	}

	for _, cluster := range clusters {
		if cluster == name {
			return true, nil
		}
	}

	return false, nil
}

func LoadCluster(name string) (*Cluster, error) {
	var cluster Cluster
	cluster.Name = name

	if err := cluster.LoadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read cluster (%v) configuration: %w", name, err)
	}

	if err := cluster.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cluster (%v) configuration: %w", name, err)
	}

	return &cluster, nil
}

func (c *Cluster) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("missing cluster name")
	}

	if c.Type != HAClusterType {
		return fmt.Errorf("unknown cluster type: %v", c.Type)
	}

	for index, name := range c.Nodes {
		node, err := LoadNode(name)
		if err != nil {
			return fmt.Errorf("failed loading node (%d / %v): %w", index, name, err)
		}

		if node.Cluster != c.Name {
			return fmt.Errorf("node (%d / %v) not listed in cluster; listed in `%v`", index, name, node.Cluster)
		}
	}

	return nil
}

func (c *Cluster) LoadConfig() error {
	directory := c.GetDirectory()
	path := filepath.Join(directory, ClusterJsonName)
	configFile, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open config file (`%v`) for reading: %w", path, err)
	}

	defer configFile.Close()

	// We need to unmarshal to an intermediate interface so that we can figure
	// out the correct types for the Storage and Listeners.
	var cfg map[string]interface{}

	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := c.FromInterface(cfg); err != nil {
		return fmt.Errorf("failed to translate config: %w", err)
	}

	return nil
}

func (c *Cluster) FromInterface(iface map[string]interface{}) error {
	c.Name = iface["name"].(string)
	c.Type = iface["type"].(string)

	if nodesRaw, ok := iface["nodes"].([]interface{}); ok {
		c.Nodes = nil
		for _, nodeRaw := range nodesRaw {
			c.Nodes = append(c.Nodes, nodeRaw.(string))
		}
	}

	return nil
}

func ClusterBaseDirectory() string {
	usr, _ := user.Current()
	dir := usr.HomeDir

	return filepath.Join(dir, ".local/share/devbao/clusters")
}

func (c *Cluster) GetDirectory() string {
	return filepath.Join(ClusterBaseDirectory(), c.Name)
}

func (c *Cluster) SaveConfig() error {
	if err := c.Validate(); err != nil {
		return fmt.Errorf("failed validating config prior to saving: %w", err)
	}

	directory := c.GetDirectory()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("failed to create cluster directory (%v): %w", directory, err)
	}

	path := filepath.Join(directory, ClusterJsonName)
	configFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open config file (`%v`) for writing: %w", path, err)
	}

	defer configFile.Close()

	if err := json.NewEncoder(configFile).Encode(c); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return nil
}

func (c *Cluster) GetLeader() (*Node, *api.Client, error) {
	for index, name := range c.Nodes {
		node, err := LoadNode(name)
		if err != nil {
			return nil, nil, fmt.Errorf("error loading node %d / %v: %w", index, name, err)
		}

		client, err := node.GetClient()
		if err != nil {
			return nil, nil, fmt.Errorf("error getting client for node %d / %v: %w", index, name, err)
		}

		resp, err := client.Sys().Leader()
		if err != nil {
			return nil, nil, fmt.Errorf("error getting leadership status for node %d / %v: %w", index, name, err)
		}

		if resp.IsSelf {
			return node, client, nil
		}
	}

	return nil, nil, fmt.Errorf("no leader found on cluster; %v nodes", len(c.Nodes))
}

func (c *Cluster) JoinNodeHACluster(node *Node) error {
	leaderNode, leaderClient, err := c.GetLeader()
	if err != nil {
		return fmt.Errorf("error finding leader: %w", err)
	}

	nodeClient, err := node.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client for node to add: %w", err)
	}

	if len(leaderNode.Config.Seals) != len(node.Config.Seals) {
		return fmt.Errorf("mismatched seal configuration counts between %v and %v; cannot join existing cluster -- ensure seals are configured correctly and retry", leaderNode.Name, node.Name)
	}

	for index, seal := range leaderNode.Config.Seals {
		leaderConfig, err := seal.ToConfig("/")
		if err != nil {
			return fmt.Errorf("error building seal config %d for %v: %w", index, leaderNode.Name, err)
		}

		followerConfig, err := node.Config.Seals[index].ToConfig("/")
		if err != nil {
			return fmt.Errorf("error building seal config %d for %v: %w", index, node.Name, err)
		}

		if leaderConfig != followerConfig {
			return fmt.Errorf("mismatched seal configuration counts between %v and %v; cannot join existing cluster -- ensure seals are configured correctly and retry", leaderNode.Name, node.Name)
		}
	}

	resp, err := nodeClient.Sys().RaftJoin(&api.RaftJoinRequest{
		LeaderAPIAddr: leaderClient.Address(),
		Retry:         true,
	})
	if err != nil {
		return fmt.Errorf("failed joining node %v to cluster %v / leader %v: %w", node.Name, c.Name, leaderNode.Name, err)
	}

	// Update this node's token to mirror the leadership.
	node.Token = leaderNode.Token
	node.UnsealKeys = leaderNode.UnsealKeys
	node.Cluster = c.Name

	if err := node.SaveConfig(); err != nil {
		return fmt.Errorf("failed saving updated state for joined node %v: %w", node.Name, err)
	}

	if !resp.Joined {
		// Attempt to unseal using stored shamir's keys.
		if _, err := node.Unseal(); err != nil {
			return fmt.Errorf("failed unsealing follower node %v: %w", node.Name, err)
		}
	}

	c.Nodes = append(c.Nodes, node.Name)
	if err := c.SaveConfig(); err != nil {
		return fmt.Errorf("failed saving updated cluster state: %w", err)
	}

	return nil
}
