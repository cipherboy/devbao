package main

import (
	"fmt"
	"os"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildNodeResumeCommand() *cli.Command {
	c := &cli.Command{
		Name:      "resume",
		Aliases:   []string{"r"},
		ArgsUsage: "<name>",
		Usage:     "resume the named instance if it is not running",

		Action: RunNodeResumeCommand,
	}

	return c
}

func RunNodeResumeCommand(cCtx *cli.Context) error {
	if !cCtx.Args().Present() {
		return fmt.Errorf("missing required positional argument: <name>, the name of the instance to resume")
	}

	name := cCtx.Args().First()
	node, err := bao.LoadNode(name)
	if err != nil {
		return err
	}

	if err := node.Exec.ValidateRunning(); err == nil {
		fmt.Fprintf(os.Stderr, "node %v / pid %v is already running\n", name, node.Exec.Pid)
		return nil
	}

	if node.Config.Dev != nil {
		fmt.Fprintf(os.Stderr, "warning: node %v is a dev mode instance; this means its storage was not persistent and will have different state\n", name)
	}

	fmt.Printf("resuming node %v...\n", name)
	return node.Resume()
}
