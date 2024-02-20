package main

import (
	"github.com/urfave/cli/v2"
)

func BuildNodeCommand() *cli.Command {
	c := &cli.Command{
		Name:    "node",
		Aliases: []string{"n"},
		Usage:   "commands for managing non-clustered nodes",
	}

	c.Subcommands = append(c.Subcommands, BuildNodeCleanCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeDirCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeEnvCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeListCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeResumeCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStartCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStartDevCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStopCommand())

	return c
}
