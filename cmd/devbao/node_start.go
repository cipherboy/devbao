package main

import (
	"fmt"
	"strings"

	"github.com/cipherboy/devbao/pkg/bao"

	"github.com/urfave/cli/v2"
)

func ServerNameFlags(name string) []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "name",
			Value: name,
			Usage: "name for the instance",
		},
	}
}

func ServerFlags(name string) []cli.Flag {
	typeFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "type",
			Value: "",
			Usage: "type of node to run: `` for auto-detect preferring OpenBao, `bao` to run an OpenBao instance, or `vault` to run a HashiCorp Vault instance.",
		},
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Value:   false,
			Usage:   "overwrite an existing node, if present",
		},
	}

	return append(ServerNameFlags(name), typeFlags...)
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

	c.Flags = append(c.Flags, ServerFlags("dev")...)
	c.Flags = append(c.Flags, DevServerFlags()...)

	return c
}

func RunNodeStartDevCommand(cCtx *cli.Context) error {
	if cCtx.Args().Present() {
		return fmt.Errorf("unexpected positional argument -- this command takes none: `%v`", cCtx.Args().First())
	}

	name := cCtx.String("name")
	nType := cCtx.String("type")
	force := cCtx.Bool("force")

	if !force {
		present, err := bao.NodeExists(name)
		if err != nil {
			return fmt.Errorf("error checking if node exists: %w", err)
		}

		if present {
			return fmt.Errorf("refusing to overwrite existing node %v", name)
		}
	}
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
		&cli.BoolFlag{
			Name:    "initialize",
			Aliases: []string{"auto-initialize", "i"},
			Value:   false,
			Usage:   "Automatically initialize the underlying node, saving unseal keys",
		},
		&cli.BoolFlag{
			Name:    "unseal",
			Aliases: []string{"auto-unseal", "u"},
			Value:   false,
			Usage:   "Automatically unseal the underlying node; requires --initialize",
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

	c.Flags = append(c.Flags, ServerFlags("prod")...)
	c.Flags = append(c.Flags, ProdServerFlags()...)

	return c
}

func RunNodeStartCommand(cCtx *cli.Context) error {
	if cCtx.Args().Present() {
		return fmt.Errorf("unexpected positional argument -- this command takes none: `%v`", cCtx.Args().First())
	}

	name := cCtx.String("name")
	nType := cCtx.String("type")
	storage := cCtx.String("storage")
	initialize := cCtx.Bool("initialize")
	unseal := cCtx.Bool("unseal")
	force := cCtx.Bool("force")

	if !force {
		present, err := bao.NodeExists(name)
		if err != nil {
			return fmt.Errorf("error checking if node exists: %w", err)
		}

		if present {
			return fmt.Errorf("refusing to overwrite existing node %v", name)
		}
	}

	if unseal && !initialize {
		return fmt.Errorf("--unseal requires --initialize, but was not provided")
	}

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

	if err := node.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	if initialize {
		if err := node.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize node: %w", err)
		}

		if unseal {
			if _, err := node.Unseal(); err != nil {
				return fmt.Errorf("failed to unseal node: %w", err)
			}
		}
	}

	return nil
}
