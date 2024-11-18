package main

import (
	_ "fmt"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	cmd "minidocker/cmd"
	"os"
)

const usage = "A simple container runtime implemented in Go"

func main() {
	app := cli.NewApp()
	app.Name = "minidocker"
	app.Usage = usage
	app.Commands = []cli.Command{
		cmd.InitCommand,
		cmd.RunCommand,
	}

	// set logger
	app.Before = func(context *cli.Context) error {
		log.SetFormatter(&log.TextFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
