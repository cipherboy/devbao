package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildNodeCleanCommand() *cli.Command {
	c := &cli.Command{
		Name:      "clean",
		Aliases:   []string{"remove", "c"},
		ArgsUsage: "<name>",
		Usage:     "remove the named instance",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Value:   false,
				Usage:   "force removal of node",
			},
		},

		Action: RunNodeCleanCommand,
	}

	return c
}

func RunNodeCleanCommand(cCtx *cli.Context) error {
	if !cCtx.Args().Present() {
		return fmt.Errorf("missing required positional argument: <name>, the name of the instance to remove")
	}

	force := cCtx.Bool("force")

	name := cCtx.Args().First()
	node, err := bao.LoadNode(name)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			fmt.Fprintf(os.Stderr, "node %v was already removed\n", name)
			return nil
		}

		// Some other unknown error.
		if !force {
			return fmt.Errorf("failed to load node to determine state: %w", err)
		} else {
			node = &bao.Node{
				Name: name,
			}
		}
	}

	if node.Exec != nil {
		if err := node.Exec.ValidateRunning(); err == nil {
			fmt.Fprintf(os.Stderr, "node %v / pid %v is running, stopping...\n", name, node.Exec.Pid)
			if err := node.Kill(); err != nil {
				return fmt.Errorf("failed to stop node prior to removal: %w", err)
			}
		}
	}

	fmt.Printf("cleaning node %v...\n", name)
	return node.Clean(force)
}
