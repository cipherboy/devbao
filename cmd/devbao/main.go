package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                   "devbao",
		Usage:                  "manage and launch OpenBao (HashiCorp Vault) instances",
		Version:                "0.0.1",
		UseShortOptionHandling: true,
		EnableBashCompletion:   true,
		Suggest:                true,
	}

	app.Commands = append(app.Commands, BuildStartDevCommand())
	app.Commands = append(app.Commands, BuildStartCommand())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
