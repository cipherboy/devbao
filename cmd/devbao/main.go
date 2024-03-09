package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                   "devbao",
		Usage:                  "manage and launch OpenBao (or HashiCorp Vault) instances",
		Version:                "0.0.1",
		UseShortOptionHandling: true,
		EnableBashCompletion:   true,
		Suggest:                true,
	}

	app.Commands = append(app.Commands, BuildClusterCommand())
	app.Commands = append(app.Commands, BuildNodeCommand())
	app.Commands = append(app.Commands, BuildProfileCommand())
	app.Commands = append(app.Commands, BuildTUICommand())

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
