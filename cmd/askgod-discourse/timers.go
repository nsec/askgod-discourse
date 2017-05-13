package main

import (
	"time"

	"gopkg.in/inconshreveable/log15.v2"
)

func (s *syncer) setupTimers() (chan error, error) {
	chError := make(chan error, 1)

	go func() {
		for {
			time.Sleep(30 * time.Second)
			s.logger.Debug("Processing timer based tasks")

			// Process pending users
			s.logger.Debug("Looking for pending users")
			err := s.discourseProcessNewUsers()
			if err != nil {
				s.logger.Error("Failed to process pending users", log15.Ctx{"error": err})
				continue
			}

			// Look for scheduled posts
			s.logger.Debug("Looking for scheduled posts")
		}
	}()

	return chError, nil
}
