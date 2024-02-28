package main

import (
	"fmt"
	"os"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildNodeUnsealCommand() *cli.Command {
	c := &cli.Command{
		Name:      "unseal",
		Aliases:   []string{"u"},
		ArgsUsage: "<name>",
		Usage:     "unseals the specified node using locally stored unseal keys",

		Action: RunNodeUnsealCommand,
	}

	return c
}

func RunNodeUnsealCommand(cCtx *cli.Context) error {
	if !cCtx.Args().Present() {
		return fmt.Errorf("missing required positional argument:\n\t<name>, the node which should be unsealed")
	}

	name := cCtx.Args().First()

	node, err := bao.LoadNode(name)
	if err != nil {
		return fmt.Errorf("failed to load node: %w", err)
	}

	client, err := node.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client for node: %w", err)
	}

	if len(node.UnsealKeys) == 0 {
		return fmt.Errorf("no unseal keys stored for node %v", name)
	}

	for index, key := range node.UnsealKeys {
		status, err := client.Sys().SealStatus()
		if err != nil {
			return fmt.Errorf("failed to fetch unseal status: %w", err)
		}

		if !status.Sealed {
			if index == 0 {
				fmt.Fprintf(os.Stderr, "[warning] node %v was already unsealed\n", name)
			}
			break
		}

		_, err = client.Sys().Unseal(key)
		if err != nil {
			return fmt.Errorf("failed to provide unseal shard: %w", err)
		}
	}

	return nil
}
