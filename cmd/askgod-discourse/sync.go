package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nsec/askgod/api"
	"gopkg.in/yaml.v2"
)

func (s *syncer) syncTeams() error {
	s.teamsLock.Lock()
	defer s.teamsLock.Unlock()

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
		dbTeamsMap[entry.AskgodID] = entry
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
		_, ok := askgodTeamsMap[entry.AskgodID]

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

type post struct {
	Type      string                      `yaml:"type"`
	Topic     string                      `yaml:"topic"`
	Trigger   *postTrigger                `yaml:"trigger"`
	Title     string                      `yaml:"title"`
	API       *postAPI                    `yaml:"api"`
	Body      string                      `yaml:"body"`
	Variables map[string]map[int64]string `yaml:"variables"`
	Posts     []*struct {
		API  *postAPI `yaml:"api"`
		Body string   `yaml:"body"`
	} `yaml:"posts"`
}

type postTrigger struct {
	Type      string `yaml:"type"`
	Tag       string `yaml:"tag"`
	Value     int64  `yaml:"value"`
	After     string `yaml:"after"`
	AfterTime time.Time
}

type postAPI struct {
	User string `yaml:"user"`
	Key  string `yaml:"key"`
}

func (s *syncer) syncPosts() error {
	s.postsLock.Lock()
	defer s.postsLock.Unlock()

	posts := map[string]post{}

	// Get the submitted flags
	askgodFlags, err := s.askgodGetTeamDiscourseFlags()
	if err != nil {
		return err
	}

	// Get the current scores
	askgodScores, err := s.askgodGetTeamScores()
	if err != nil {
		return err
	}

	// Get all the posts
	dbTeamPosts, err := s.dbGetTeamPosts()
	if err != nil {
		return err
	}

	// Get all the teams from the database
	dbTeams, err := s.dbGetTeams()
	if err != nil {
		return err
	}

	// Enumerate the posts directory
	files, err := ioutil.ReadDir(s.config.Posts)
	if err != nil {
		return err
	}

	// Parse the individual yaml files
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		// Get the full path
		path := filepath.Join(s.config.Posts, file.Name())

		// Read the file
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse the content
		newPost := post{}
		err = yaml.Unmarshal(content, &newPost)
		if err != nil {
			return fmt.Errorf("Failed to parse '%s': %v", path, err)
		}

		// Convert timestamps
		if newPost.Trigger != nil {
			if newPost.Trigger.After != "" {
				ts, err := time.ParseInLocation("2006/01/02 15:04", newPost.Trigger.After, time.Local)
				if err != nil {
					return err
				}

				newPost.Trigger.AfterTime = ts
			}
		}

		// Add the post to the map
		name := strings.TrimSuffix(file.Name(), ".yaml")
		posts[name] = newPost
	}

	// Processing of post entries
	processEntry := func(postType string) error {
		for name, post := range posts {
			teams := []dbTeam{}

			// Sort out API keys
			apiUser := s.config.DiscourseAPIUser
			apiKey := s.config.DiscourseAPIKey
			if post.API != nil {
				apiUser = post.API.User
				apiKey = post.API.Key
			}

			// Only process the type we've been asked for
			if post.Type != postType {
				continue
			}

			// Validate the trigger
			if post.Trigger != nil {
				if post.Trigger.Type == "timer" {
					if post.Trigger.AfterTime.Unix() > time.Now().Unix() {
						// Not time yet
						continue
					}

					// If it's time, send to everyone
					teams = dbTeams
				} else if post.Trigger.Type == "flag" {
					for _, team := range dbTeams {
						if post.Trigger.Tag == "" {
							if askgodScores[team.AskgodID] == 0 {
								// Hasn't sent a flag yet
								continue
							}
						} else {
							if !int64InSlice(team.AskgodID, askgodFlags[post.Trigger.Tag]) {
								// Not scored that yet
								continue
							}
						}
						teams = append(teams, team)
					}
				} else if post.Trigger.Type == "score" {
					for _, team := range dbTeams {
						if askgodScores[team.AskgodID] < post.Trigger.Value {
							// Not there yet
							continue
						}

						teams = append(teams, team)
					}
				}
			} else {
				// Everyone is getting the post
				teams = dbTeams
			}

			// Post to affected teams
			for _, team := range teams {
				if len(s.config.PublishRestricted) > 0 && !stringInSlice(team.DiscourseName, s.config.PublishRestricted) {
					continue
				}

				_, ok := dbTeamPosts[team.AskgodID][name]
				if ok {
					// Already posted for this team, skip
					continue
				}

				// Apply templating
				if team.AskgodName == "" {
					team.AskgodName = team.DiscourseName
				}

				post.Body = strings.Replace(post.Body, "%{team_name}", team.AskgodName, -1)
				post.Body = strings.Replace(post.Body, "%{team_score}", fmt.Sprintf("%d", askgodScores[team.AskgodID]), -1)

				// Process template variables
				r := regexp.MustCompile(`%\{(\w+)\}`)
				post.Body = r.ReplaceAllStringFunc(post.Body, func(p string) string {
					return post.Variables[p[2:len(p)-1]][team.AskgodID]
				})

				if post.Type == "topic" {
					err := s.discourseCreateTopic(team.DiscourseName, team.AskgodID, apiUser, apiKey, name, team.DiscourseCategoryID, post.Title, post.Body)
					if err != nil {
						return err
					}
				} else if post.Type == "post" {
					postID := dbTeamPosts[team.AskgodID][post.Topic]
					err := s.discourseCreatePost(team.DiscourseName, team.AskgodID, apiUser, apiKey, name, postID, post.Body)
					if err != nil {
						return err
					}
				} else if post.Type == "posts" {
					postID := dbTeamPosts[team.AskgodID][post.Topic]
					for _, subPost := range post.Posts {
						subApiUser := apiUser
						subApiKey := apiKey
						if post.API != nil {
							subApiUser = subPost.API.User
							subApiKey = subPost.API.Key
						}

						err := s.discourseCreatePost(team.DiscourseName, team.AskgodID, subApiUser, subApiKey, name, postID, subPost.Body)
						if err != nil {
							return err
						}
					}
				} else {
					return fmt.Errorf("Invalid type: %s", post.Type)
				}
			}
		}

		return nil
	}

	// Process all topics first
	err = processEntry("topic")
	if err != nil {
		return err
	}

	// Delete removed posts
	for _, entry := range dbTeamPosts {
		for name, postid := range entry {
			_, err := os.Lstat(filepath.Join(s.config.Posts, fmt.Sprintf("%s.yaml", name)))
			if err != nil && os.IsNotExist(err) {
				err = s.discourseDeleteTopic(postid)
				if err != nil {
					return err
				}

				err = s.dbDeletePost(postid)
				if err != nil {
					return err
				}
			}
		}
	}

	// Refresh the list of posts
	dbTeamPosts, err = s.dbGetTeamPosts()
	if err != nil {
		return err
	}

	// Then the posts
	err = processEntry("post")
	if err != nil {
		return err
	}

	// Then the posts
	err = processEntry("posts")
	if err != nil {
		return err
	}

	return nil
}
