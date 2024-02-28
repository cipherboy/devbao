package main

import (
	"fmt"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func BuildProfileListCommand() *cli.Command {
	c := &cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Usage:   "list available configuration profiles",

		Action: RunProfileListCommand,
	}

	return c
}

func RunProfileListCommand(cCtx *cli.Context) error {
	for _, policy := range bao.ListPolicies() {
		fmt.Printf(" - %v\n", policy)
	}
	return nil
}
