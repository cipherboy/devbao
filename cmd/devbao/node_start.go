package main

import (
	"fmt"
	"strings"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func ServerFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Value: "dev",
			Usage: "name for the instance",
		},
		&cli.StringFlag{
			Name:  "type",
			Value: "",
			Usage: "type of node to run: `` for auto-detect preferring OpenBao, `bao` to run an OpenBao instance, or `vault` to run a HashiCorp Vault instance.",
		},
	}
}

func DevServerFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "token",
			Value: "devroot",
			Usage: "development mode root token identifier",
		},
		&cli.StringFlag{
			Name:  "address",
			Value: "0.0.0.0:8200",
			Usage: "development mode listener bind address",
		},
	}
}

func BuildNodeStartDevCommand() *cli.Command {
	c := &cli.Command{
		Name:    "start-dev",
		Aliases: []string{"d"},
		Usage:   "start a dev-mode instance",

		Action: RunNodeStartDevCommand,
	}

	c.Flags = append(c.Flags, ServerFlags()...)
	c.Flags = append(c.Flags, DevServerFlags()...)

	return c
}

func RunNodeStartDevCommand(cCtx *cli.Context) error {
	name := cCtx.String("name")
	nType := cCtx.String("type")

	opts := &bao.DevConfig{
		Token:   cCtx.String("token"),
		Address: cCtx.String("address"),
	}

	node, err := bao.BuildNode(name, nType, opts)
	if err != nil {
		return fmt.Errorf("failed to build node: %w", err)
	}

	return node.Start()
}

func ProdServerFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "listeners",
			Value: "tcp:0.0.0.0:8200",
			Usage: "Bind address of the listener to add, separated by commas for multiple.\nUse `tcp:` to prefix network listener bind addresses or `unix:` to prefix socket listener paths.",
		},
		&cli.StringFlag{
			Name:  "storage",
			Value: "raft",
			Usage: "Storage backend to use; choose between `raft`, `file`, or `inmem`. File and Memory backends are not recommended for production use.",
		},
	}
}

func BuildNodeStartCommand() *cli.Command {
	c := &cli.Command{
		Name:    "start",
		Aliases: []string{"s"},
		Usage:   "start a production instance",

		Action: RunNodeStartCommand,
	}

	c.Flags = append(c.Flags, ServerFlags()...)
	c.Flags = append(c.Flags, ProdServerFlags()...)

	return c
}

func RunNodeStartCommand(cCtx *cli.Context) error {
	name := cCtx.String("name")
	nType := cCtx.String("type")
	storage := cCtx.String("storage")

	var opts []bao.NodeConfigOpt

	switch storage {
	case "", "raft":
		opts = append(opts, &bao.RaftStorage{})
	case "file":
		opts = append(opts, &bao.FileStorage{})
	case "inmem":
		opts = append(opts, &bao.InmemStorage{})
	default:
		return fmt.Errorf("unknown value for -storage: `%v`; supported values are `raft`, `file`, or `inmem`", storage)
	}

	listeners := strings.Split(cCtx.String("listeners"), ",")
	for index, listener := range listeners {
		if strings.HasPrefix(listener, "tcp:") {
			opts = append(opts, &bao.TCPListener{
				Address: strings.TrimPrefix(listener, "tcp:"),
			})
		} else if strings.HasPrefix(listener, "unix:") {
			opts = append(opts, &bao.UnixListener{
				Path: strings.TrimPrefix(listener, "unix:"),
			})
		} else {
			return fmt.Errorf("unknown type prefix for -listeners at index %d: `%v`; supported values are `tcp:<bind address>` or `unix:<path>`", index, listener)
		}
	}

	node, err := bao.BuildNode(name, nType, opts...)
	if err != nil {
		return fmt.Errorf("failed to build node: %w", err)
	}

	return node.Start()
}
