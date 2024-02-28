package main

import (
	"github.com/urfave/cli/v2"
)

func BuildProfileCommand() *cli.Command {
	c := &cli.Command{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "commands for provisioning a usage profile on a node",
	}

	c.Subcommands = append(c.Subcommands, BuildProfileApplyCommand())
	c.Subcommands = append(c.Subcommands, BuildProfileListCommand())

	return c
}
