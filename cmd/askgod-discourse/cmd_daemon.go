package main

import (
	"gopkg.in/urfave/cli.v1"
)

func cmdDaemon(ctx *cli.Context) error {
	// Load configuration
	s, err := getSyncer("config.yaml")
	if err != nil {
		return err
	}

	// Connect to the DB
	err = s.dbSetup()
	if err != nil {
		return err
	}

	// Setup event handlers
	s.logger.Info("Setting up events")
	chEvents, err := s.setupEvents()
	if err != nil {
		return err
	}

	// Setup timers
	s.logger.Info("Setting up timers")
	chTimers, err := s.setupTimers()
	if err != nil {
		return err
	}

	// Process backlog
	s.logger.Info("Running initial team sync")
	err = s.syncTeams()
	if err != nil {
		return err
	}

	s.logger.Info("Running initial posts sync")
	err = s.syncPosts()
	if err != nil {
		return err
	}

	s.logger.Info("Running initial account approval")
	err = s.discourseProcessNewUsers()
	if err != nil {
		return err
	}

	// Wait for something to fail
	select {
	case err := <-chEvents:
		if err != nil {
			return err
		}
	case err := <-chTimers:
		if err != nil {
			return err
		}
	}

	return nil
}
