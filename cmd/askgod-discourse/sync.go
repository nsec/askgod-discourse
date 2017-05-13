package main

import (
	"github.com/nsec/askgod/api"
)

func (s *syncer) syncTeams() error {
	// Get all teams from askgod
	askgodTeams, err := s.askgodGetTeams()
	if err != nil {
		return err
	}

	// Make a map based on askgod team id
	askgodTeamsMap := map[int64]api.AdminTeam{}
	for _, entry := range askgodTeams {
		askgodTeamsMap[entry.ID] = entry
	}

	// Get all teams from DB
	dbTeams, err := s.dbGetTeams()
	if err != nil {
		return err
	}

	// Make a map based on askgod team id
	dbTeamsMap := map[int64]dbTeam{}
	for _, entry := range dbTeams {
		dbTeamsMap[entry.ID] = entry
	}

	// Update the teams
	for _, entry := range askgodTeams {
		dbEntry, ok := dbTeamsMap[entry.ID]

		// New team
		if !ok {
			discourseName := entry.Tags["discourse"]
			if discourseName == "" {
				continue
			}

			// Create the team
			err := s.discourseCreateTeam(discourseName, entry.ID, entry.Name)
			if err != nil {
				return err
			}

			continue
		}

		// Existing team
		if entry.Name != dbEntry.AskgodName {
			// Rename the team
			err := s.discourseRenameTeam(dbEntry.DiscourseName, dbEntry.DiscourseGroupID, entry.Name)
			if err != nil {
				return err
			}
		}
	}

	// Delete removed teams
	for _, entry := range dbTeams {
		_, ok := askgodTeamsMap[entry.ID]

		if !ok {
			// Delete the team
			err := s.discourseDeleteTeam(entry.DiscourseName, entry.DiscourseGroupID, entry.DiscourseCategoryID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *syncer) syncPosts() error {
	return nil
}
