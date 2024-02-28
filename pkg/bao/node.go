package bao

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/openbao/openbao/api"
)

const (
	NODE_JSON_NAME       = "node.json"
	INSTANCE_CONFIG_NAME = "config.hcl"
)

type Node struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Exec   *ExecEnvironment `json:"exec"`
	Config NodeConfig       `json:"config"`

	Token      string   `json:"token"`
	UnsealKeys []string `json:"unseal_keys,omitempty"`
}

func (n *Node) FromInterface(iface map[string]interface{}) error {
	n.Name = iface["name"].(string)
	n.Type = iface["type"].(string)

	if _, present := iface["token"]; present {
		n.Token = iface["token"].(string)
	}

	if unsealKeysRaw, ok := iface["unseal_keys"].([]interface{}); ok {
		n.UnsealKeys = nil
		for _, keyRaw := range unsealKeysRaw {
			n.UnsealKeys = append(n.UnsealKeys, keyRaw.(string))
		}
	}

	data, present := iface["exec"]
	if present {
		j, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("error re-marshalling exec config: %w", err)
		}

		if err := json.Unmarshal(j, &n.Exec); err != nil {
			return fmt.Errorf("error unmarshaling exec config; %w", err)
		}
	}

	return n.Config.FromInterface(iface["config"].(map[string]interface{}))
}

type NodeConfigOpt interface{}

var (
	_ NodeConfigOpt = Listener(nil)
	_ NodeConfigOpt = Storage(nil)
	_ NodeConfigOpt = &DevConfig{}
)

func ListNodes() ([]string, error) {
	dir := NodeBaseDirectory()
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Errorf("error listing node directory (`%v`): %w", dir, err)
	}

	var results []string
	for _, entry := range entries {
		if entry.IsDir() {
			results = append(results, entry.Name())
		}
	}

	return results, nil
}

func LoadNode(name string) (*Node, error) {
	var node Node
	node.Name = name

	if err := node.LoadConfig(); err != nil {
		return nil, fmt.Errorf("failed to read node (%v) configuration: %w", name, err)
	}

	if err := node.Validate(); err != nil {
		return nil, fmt.Errorf("invalid node (%v) configuration: %w", name, err)
	}

	return &node, nil
}

func BuildNode(name string, product string, opts ...NodeConfigOpt) (*Node, error) {
	n := &Node{
		Name: name,
		Type: product,
	}

	for index, opt := range opts {
		switch tOpt := opt.(type) {
		case Listener:
			n.Config.Listeners = append(n.Config.Listeners, tOpt)
		case Storage:
			n.Config.Storage = tOpt
		case *DevConfig:
			n.Config.Dev = tOpt
		default:
			return nil, fmt.Errorf("unknown type of node configuration option at index %d: %v (%T)", index, opt, opt)
		}
	}

	if err := n.Validate(); err != nil {
		return nil, fmt.Errorf("invalid node configuration: %w", err)
	}

	return n, nil
}

func (n *Node) Validate() error {
	if n.Name == "" {
		if n.Config.Dev == nil {
			n.Name = "dev"
		} else {
			n.Name = "prod"
		}
	}

	if n.Type != "" && n.Type != "bao" && n.Type != "vault" {
		return fmt.Errorf("invalid node type (`%s`): expected either empty (``), OpenBao (`bao`), or HashiCorp Vault (`vault`)", n.Type)
	}

	if n.Config.Dev != nil && n.Token == "" && n.Config.Dev.Token != "" {
		n.Token = n.Config.Dev.Token
	}

	return n.Config.Validate()
}

func NodeBaseDirectory() string {
	usr, _ := user.Current()
	dir := usr.HomeDir

	return filepath.Join(dir, ".local/share/devbao/nodes")
}

func (n *Node) GetDirectory() string {
	return filepath.Join(NodeBaseDirectory(), n.Name)
}

func (n *Node) buildExec() error {
	if err := n.Validate(); err != nil {
		return fmt.Errorf("failed to validate node definition: %w", err)
	}

	directory := n.GetDirectory()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("failed to create node directory (%v): %w", directory, err)
	}

	addr, _, err := n.Config.GetConnectAddr()
	if err != nil {
		return fmt.Errorf("failed to get connection address for node %v: %w", n.Name, err)
	}

	args, err := n.Config.AddArgs(directory)
	if err != nil {
		return fmt.Errorf("failed to build arguments to binary: %w", err)
	}

	config, err := n.Config.ToConfig(directory)
	if err == nil && config == "" && n.Config.Dev == nil {
		err = fmt.Errorf("expected non-dev server to have non-empty configuration; are listeners or storage missing")
	}
	if err != nil {
		return fmt.Errorf("failed to build node's configuration (%s): %w", n.Name, err)
	}

	if config != "" {
		path, err := n.SaveInstanceConfig(config)
		if err != nil {
			return fmt.Errorf("error persisting node configuration: %w", err)
		}

		args = append(args, "-config="+path)
	}

	n.Exec = &ExecEnvironment{
		Args:      args,
		Directory: directory,

		ConnectAddress: addr,
	}

	return nil
}

func (n *Node) Start() error {
	_ = n.Kill()

	if err := n.Clean(); err != nil {
		return fmt.Errorf("failed to clean up existing node: %w", err)
	}

	return n.Resume()
}

func (n *Node) Kill() error {
	if n.Exec == nil || n.Exec.Pid == 0 {
		disk, err := LoadNode(n.Name)
		if err != nil {
			return fmt.Errorf("error loading node from disk while killing: %w", err)
		}

		return disk.Exec.Kill()
	}

	return n.Exec.Kill()
}

func (n *Node) Clean() error {
	directory := n.GetDirectory()
	return os.RemoveAll(directory)
}

func (n *Node) Resume() error {
	if err := n.buildExec(); err != nil {
		return fmt.Errorf("failed to build execution environment: %w", err)
	}

	var err error
	switch n.Type {
	case "":
		err = Exec(n.Exec)
	case "bao":
		err = ExecBao(n.Exec)
	case "vault":
		err = ExecVault(n.Exec)
	default:
		err = fmt.Errorf("unknown execution type: `%s`", n.Type)
	}

	if err != nil {
		return err
	}

	// Now, persist node configuration.
	return n.SaveConfig()
}

func (n *Node) LoadConfig() error {
	directory := n.GetDirectory()
	path := filepath.Join(directory, NODE_JSON_NAME)
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

	if err := n.FromInterface(cfg); err != nil {
		return fmt.Errorf("failed to translate config: %w", err)
	}

	return nil
}

func (n *Node) SaveConfig() error {
	if err := n.Validate(); err != nil {
		return fmt.Errorf("failed validating config prior to saving: %w", err)
	}

	directory := n.GetDirectory()
	path := filepath.Join(directory, NODE_JSON_NAME)
	configFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open config file (`%v`) for writing: %w", path, err)
	}

	defer configFile.Close()

	if err := json.NewEncoder(configFile).Encode(n); err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return nil
}

func (n *Node) SaveInstanceConfig(config string) (string, error) {
	directory := n.GetDirectory()
	path := filepath.Join(directory, INSTANCE_CONFIG_NAME)

	configFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", fmt.Errorf("failed to open instance config file (`%v`) for writing: %w", path, err)
	}

	defer configFile.Close()

	if _, err := io.WriteString(configFile, config); err != nil {
		return "", fmt.Errorf("failed to write instance config file (`%v`): %w", path, err)
	}

	return path, nil
}

func (n *Node) GetConnectAddr() (string, error) {
	addr, isTls, err := n.Config.GetConnectAddr()
	if err != nil {
		return "", fmt.Errorf("failed to get connection address for node %v: %w", n.Name, err)
	}

	scheme := "http"
	if isTls {
		scheme = "https"
	}

	return fmt.Sprintf("%v://%v", scheme, addr), nil
}

func (n *Node) GetEnv() (map[string]string, error) {
	results := make(map[string]string)
	prefix := "VAULT_"

	addr, err := n.GetConnectAddr()
	if err != nil {
		return nil, err
	}

	results[prefix+"ADDR"] = addr
	results[prefix+"TOKEN"] = n.Token

	return results, nil
}

func (n *Node) GetClient() (*api.Client, error) {
	addr, err := n.GetConnectAddr()
	if err != nil {
		return nil, err
	}

	client, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		return nil, err
	}

	client.SetToken(n.Token)
	return client, nil
}

func (n *Node) SetToken(token string) (bool /* validated */, error) {
	n.Token = token

	if err := n.Exec.ValidateRunning(); err == nil {
		if err := n.ValidateToken(token); err != nil {
			return false, fmt.Errorf("token validation failed: %w", err)
		}

		return true, nil
	}

	return false, nil
}

func (n *Node) ValidateToken(token string) error {
	client, err := n.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	client.SetToken(token)

	_, err = client.Sys().ListMounts()
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	return nil
}

func (n *Node) Initialize() error {
	client, err := n.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client for node: %w", err)
	}

	initialized, err := client.Sys().InitStatus()
	if err != nil {
		return fmt.Errorf("failed to read initialization status of node: %w", err)
	}

	if initialized {
		return fmt.Errorf("node is already initialized")
	}

	if n.Token != "" || len(n.UnsealKeys) != 0 {
		return fmt.Errorf("refusing to overwrite existing token or unseal keys")
	}

	req := &api.InitRequest{
		SecretShares:    3,
		SecretThreshold: 2,
	}
	resp, err := client.Sys().Init(req)
	if err != nil {
		if !strings.Contains(err.Error(), "not applicable") {
			return fmt.Errorf("failed to initialize specified node via Shamir's: %w", err)
		}

		req = &api.InitRequest{
			RecoveryShares:    3,
			RecoveryThreshold: 2,
		}

		resp, err = client.Sys().Init(req)
		if err != nil {
			return fmt.Errorf("failed to initialize specified node via auto-unseal: %w", err)
		}
	}

	if len(resp.KeysB64) != 0 && len(resp.RecoveryKeysB64) != 0 {
		return fmt.Errorf("both unseal keys (%v) and recovery keys (%v) were provided in the response", resp.KeysB64, resp.RecoveryKeysB64)
	}

	if len(resp.KeysB64) != 0 {
		n.UnsealKeys = resp.KeysB64
	} else if len(resp.RecoveryKeysB64) != 0 {
		n.UnsealKeys = resp.RecoveryKeysB64
	}

	n.Token = resp.RootToken

	return n.SaveConfig()
}
