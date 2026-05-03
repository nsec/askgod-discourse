// Package main implements askgod-discourse, a daemon that mirrors askgod
// teams, scores and event triggers into a Discourse forum.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:                  "askgod-discourse",
		Usage:                 "Starts a daemon that processes events as they arrive",
		ArgsUsage:             "<config>",
		HideVersion:           true,
		HideHelp:              true,
		EnableShellCompletion: true,
		Action:                cmdDaemon,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}
