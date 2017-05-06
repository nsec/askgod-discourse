package main

import (
	"encoding/json"

	"github.com/nsec/askgod/api"
	"gopkg.in/inconshreveable/log15.v2"
)

func (s *syncer) setupEvents() (chan error, error) {
	chError := make(chan error, 1)

	// Websocket connection
	conn, err := s.websocket("askgod", "/events?type=flags,timeline")
	if err != nil {
		return nil, err
	}

	// Event handler
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				// Got disconnected
				chError <- err
				return
			}

			event := api.Event{}
			err = json.Unmarshal(data, &event)
			if err != nil {
				s.logger.Error("Bad askgod event", log15.Ctx{"error": err})
				continue
			}

			s.logger.Debug("Received askgod event", log15.Ctx{"type": event.Type})

			if event.Type == "flags" {
				// Got a flag submission event
				entry := api.EventFlag{}
				err = json.Unmarshal(event.Metadata, &entry)
				if err != nil {
					s.logger.Error("Bad askgod flag event", log15.Ctx{"error": err})
					continue
				}

				// We only care about valid flags
				if entry.Type != "valid" {
					continue
				}

				// Update in-memory copy
				// FIXME

				// Update discourse
				s.logger.Debug("Askgod triggered posts update")
				err = s.syncPosts()
				if err != nil {
					s.logger.Error("Failed to sync teams", log15.Ctx{"error": err})
					continue
				}
			} else if event.Type == "timeline" {
				// Got a timeline event
				entry := api.EventTimeline{}
				err = json.Unmarshal(event.Metadata, &entry)
				if err != nil {
					s.logger.Error("Bad askgod timeline event", log15.Ctx{"error": err})
					continue
				}

				// We only care about team events
				if entry.Type != "team-added" && entry.Type != "team-removed" && entry.Type != "team-updated" {
					continue
				}

				// Update in-memory copy
				// FIXME

				// Update discourse
				s.logger.Debug("Askgod triggered teams update", log15.Ctx{"type": entry.Type})
				err = s.syncTeams()
				if err != nil {
					s.logger.Error("Failed to sync teams", log15.Ctx{"error": err})
					continue
				}
			}
		}
	}()

	return chError, nil
}
