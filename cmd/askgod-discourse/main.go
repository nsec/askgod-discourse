package main

import (
	"fmt"
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "askgod-discourse"
	app.Usage = "CTF scoring system - discourse sync"
	app.HideVersion = true
	app.HideHelp = true
	app.EnableBashCompletion = true
	app.Action = cmdDaemon
	app.Usage = "Starts a daemon that processes events as they arrive"
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
	}
}
