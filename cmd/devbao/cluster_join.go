package main

import (
	"fmt"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildClusterJoinCommand() *cli.Command {
	c := &cli.Command{
		Name:      "join",
		Aliases:   []string{"j"},
		ArgsUsage: "<cluster-name> <node-name>",
		Usage:     "join a given node to the cluster",

		Action: RunClusterJoinCommand,
	}

	c.Flags = append(c.Flags, ClusterFlags()...)

	return c
}

func RunClusterJoinCommand(cCtx *cli.Context) error {
	if cCtx.Args().Len() != 2 {
		return fmt.Errorf("missing required positional argument:\n\t<cluster-name>, the name of the cluster to extend\n\t<node-name> the name of the node to join to the cluster")
	}

	clusterName := cCtx.Args().First()
	nodeName := cCtx.Args().Get(1)

	cluster, err := bao.LoadCluster(clusterName)
	if err != nil {
		return fmt.Errorf("error loading cluster: %w", err)
	}

	node, err := bao.LoadNode(nodeName)
	if err != nil {
		return fmt.Errorf("error loading node: %w", err)
	}

	if node.Cluster != "" {
		return fmt.Errorf("refusing to add node already in a cluster (`%v`)", node.Cluster)
	}

	if err := cluster.JoinNodeHACluster(node); err != nil {
		return fmt.Errorf("failed to join node to cluster: %w", err)
	}

	return nil
}
