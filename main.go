package main

import (
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

	app.Commands = []cli.Command{
		{
			Name:   "daemon",
			Usage:  "Process events as they arrive",
			Action: cmdDaemon,
		},
		{
			Name:   "sync",
			Usage:  "Get the current state and sync everything",
			Action: cmdSync,
		},
		{
			Name:      "trigger",
			Usage:     "Manually trigger a post",
			ArgsUsage: "<post name>",
			Action:    cmdTrigger,
		},
	}

	app.Run(os.Args)
}
