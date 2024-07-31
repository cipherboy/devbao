package main

import (
	"github.com/urfave/cli/v2"
)

func BuildNodeCommand() *cli.Command {
	c := &cli.Command{
		Name:    "node",
		Aliases: []string{"nodes", "n"},
		Usage:   "commands for managing individual nodes",
	}

	c.Subcommands = append(c.Subcommands, BuildNodeCleanCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeDirCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeEnvCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeGetTokenCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeGetUnsealCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeInitializeCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeListCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeResumeCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeSealCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeSetAddressCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeSetTokenCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeSetUnsealCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStartCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStartDevCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeStopCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeTailCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeTailAuditCommand())
	c.Subcommands = append(c.Subcommands, BuildNodeUnsealCommand())

	return c
}
