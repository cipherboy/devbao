package main

import (
	"fmt"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildProfileApplyCommand() *cli.Command {
	c := &cli.Command{
		Name:      "apply",
		Aliases:   []string{"a"},
		ArgsUsage: "<name> <profile>",
		Usage:     "apply configuration to a given instance",

		Action: RunProfileApplyCommand,
	}

	return c
}

func RunProfileApplyCommand(cCtx *cli.Context) error {
	if len(cCtx.Args().Slice()) != 2 {
		return fmt.Errorf("missing required positional argument: instance name and policy\nUsage: devbao policy apply <name> <position>")
	}

	name := cCtx.Args().First()
	policy := cCtx.Args().Get(1)

	node, err := bao.LoadNode(name)
	if err != nil {
		return fmt.Errorf("failed to load node: %w", err)
	}

	client, err := node.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get client for node %v: %w", name, err)
	}

	return bao.PolicySetup(client, policy)
}
