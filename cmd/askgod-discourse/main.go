// Package main implements askgod-discourse, a daemon that mirrors askgod
// teams, scores and event triggers into a Discourse forum.
package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "askgod-discourse"
	app.Usage = "CTF scoring system - discourse sync"
	app.ArgsUsage = "<config>"
	app.HideVersion = true
	app.HideHelp = true
	app.EnableBashCompletion = true
	app.Action = cmdDaemon
	app.Usage = "Starts a daemon that processes events as they arrive"
	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
