package main

import (
	"gopkg.in/urfave/cli.v1"
)

func cmdSync(ctx *cli.Context) error {
	// Setup the struct
	s, err := getSyncer("config.yaml")
	if err != nil {
		return err
	}

	// Create any missing group or categories
	err = s.discourseCreateTeams()
	if err != nil {
		return err
	}

	// Update all new users
	err = s.discourseProcessNewUsers()
	if err != nil {
		return err
	}

	return nil
}
