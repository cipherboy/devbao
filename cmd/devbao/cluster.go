package main

import (
	"github.com/urfave/cli/v2"
)

func BuildClusterCommand() *cli.Command {
	c := &cli.Command{
		Name:    "cluster",
		Aliases: []string{"clusters", "c"},
		Usage:   "commands for managing clusters",
	}

	c.Subcommands = append(c.Subcommands, BuildClusterBuildCommand())
	c.Subcommands = append(c.Subcommands, BuildClusterListCommand())

	return c
}
