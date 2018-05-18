package main

import (
	"database/sql"

	"github.com/mattn/go-sqlite3"
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

func enableForeignKeys(conn *sqlite3.SQLiteConn) error {
	_, err := conn.Exec("PRAGMA foreign_keys=ON;", nil)
	return err
}

func init() {
	sql.Register("sqlite3_with_fk", &sqlite3.SQLiteDriver{ConnectHook: enableForeignKeys})
}

// Connect sets up the database connection and returns a DB struct
func (s *syncer) dbSetup() error {
	s.logger.Info("Connecting to the database")

	sqlDB, err := sql.Open("sqlite3_with_fk", s.config.Database)
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

func (s *syncer) dbDeletePost(discoursePostID int64) error {
	// Delete a team DB entry
	_, err := s.db.Exec("DELETE FROM posts WHERE discourse_post_id=?;", discoursePostID)
	if err != nil {
		return err
	}

	return nil
}

func (s *syncer) dbGetTeamPosts() (map[int64]map[string]int64, error) {
	// Return a map of askgod teamids to map of post to postid
	resp := map[int64]map[string]int64{}

	// Fetch the needed data
	rows, err := s.db.Query("SELECT teams.askgod_id, posts.name, posts.discourse_post_id FROM posts LEFT JOIN teams ON teams.id=posts.team_id;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate through the results
	for rows.Next() {
		teamid := int64(-1)
		name := ""
		postid := int64(-1)

		err := rows.Scan(&teamid, &name, &postid)
		if err != nil {
			return nil, err
		}

		if resp[teamid] == nil {
			resp[teamid] = map[string]int64{}
		}
		resp[teamid][name] = postid
	}

	// Check for any error that might have happened
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *syncer) dbCreatePost(askgodID int64, postName string, postID int64) error {
	_, err := s.db.Exec("INSERT INTO posts (team_id, name, discourse_post_id) VALUES ((SELECT id FROM teams WHERE askgod_id=?), ?, ?);",
		askgodID, postName, postID)
	if err != nil {
		return err
	}

	return nil
}
