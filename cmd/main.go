package main

import (
	"errors"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	var app *cli.App = &cli.App{
		Name: "Message Broker. A Raft Based Implementation",
		Commands: []*cli.Command{
			{
				Name:  "gateway",
				Usage: "starts gateway server",
				Action: func(ctx *cli.Context) error {
					return errors.New("")
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
