package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const schema string = `
CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    askgod_id INTEGER NOT NULL,
    askgod_name TEXT,
    discourse_name TEXT,
    discourse_group_id INTEGER NOT NULL,
    discourse_category_id INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name TEXT,
    team_id INTEGER NOT NULL,
    discourse_post_id INTEGER NOT NULL,
    FOREIGN KEY(team_id) REFERENCES teams (id) ON DELETE CASCADE
);
`

type dbTeam struct {
	ID                  int64
	AskgodID            int64
	AskgodName          string
	DiscourseName       string
	DiscourseGroupID    int64
	DiscourseCategoryID int64
}

// Connect sets up the database connection and returns a DB struct
func (s *syncer) dbSetup() error {
	s.logger.Info("Connecting to the database")

	sqlDB, err := sql.Open("sqlite3", s.config.Database)
	if err != nil {
		return err
	}

	// Setup the DB struct
	s.db = sqlDB

	// We don't want multiple clients during setup
	s.db.SetMaxOpenConns(1)

	// Test the connection
	err = s.db.Ping()
	if err != nil {
		return err
	}

	// Create the DB schema (if needed)
	_, err = s.db.Exec(schema)
	if err != nil {
		return err
	}

	// Set the connection limit for the DB pool
	s.db.SetMaxOpenConns(10)

	return nil
}

func (s *syncer) dbGetTeams() ([]dbTeam, error) {
	// Return a list of teams
	resp := []dbTeam{}

	// Query all the teams from the database
	rows, err := s.db.Query("SELECT id, askgod_id, askgod_name, discourse_name, discourse_group_id, discourse_category_id FROM teams ORDER BY id ASC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate through the results
	for rows.Next() {
		row := dbTeam{}

		err := rows.Scan(&row.ID, &row.AskgodID, &row.AskgodName, &row.DiscourseName, &row.DiscourseGroupID, &row.DiscourseCategoryID)
		if err != nil {
			return nil, err
		}

		resp = append(resp, row)
	}

	// Check for any error that might have happened
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *syncer) dbCreateTeam(askgodID int64, askgodName string, discourseName string, discourseGroupID int64, discourseCategoryID int64) error {
	// Create a team DB entry
	_, err := s.db.Exec("INSERT INTO teams (askgod_id, askgod_name, discourse_name, discourse_group_id, discourse_category_id) VALUES (?, ?, ?, ?, ?);",
		askgodID, askgodName, discourseName, discourseGroupID, discourseCategoryID)
	if err != nil {
		return err
	}

	return nil
}

func (s *syncer) dbRenameTeam(discourseGroupID int64, askgodName string) error {
	// Change the askgod name on record
	_, err := s.db.Exec("UPDATE teams SET askgod_name=? WHERE discourse_group_id=?;", askgodName, discourseGroupID)
	if err != nil {
		return err
	}

	return nil
}

func (s *syncer) dbDeleteTeam(discourseName string, discourseGroupID int64, discourseCategoryID int64) error {
	// Delete a team DB entry
	_, err := s.db.Exec("DELETE FROM teams WHERE discourse_name=? AND discourse_group_id=? AND discourse_category_id=?;", discourseName, discourseGroupID, discourseCategoryID)
	if err != nil {
		return err
	}

	return nil
}
