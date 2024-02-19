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

	c.Subcommands = append(c.Subcommands, BuildNodeStartDevCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStartCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeListCommand())

	return c
}
