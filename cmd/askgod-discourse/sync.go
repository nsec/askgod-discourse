package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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

type postContext struct {
	posts        map[string]post
	dbTeams      []dbTeam
	dbTeamPosts  map[int64]map[string][]int64
	askgodScores map[int64]int64
	askgodFlags  map[string][]int64
}

func (s *syncer) syncPosts() error {
	s.postsLock.Lock()
	defer s.postsLock.Unlock()

	pctx := postContext{}

	// Get the submitted flags
	askgodFlags, err := s.askgodGetTeamDiscourseFlags()
	if err != nil {
		return err
	}

	pctx.askgodFlags = askgodFlags

	// Get the current scores
	askgodScores, err := s.askgodGetTeamScores()
	if err != nil {
		return err
	}

	pctx.askgodScores = askgodScores

	// Get all the posts
	dbTeamPosts, err := s.dbGetTeamPosts()
	if err != nil {
		return err
	}

	pctx.dbTeamPosts = dbTeamPosts

	// Get all the teams from the database
	dbTeams, err := s.dbGetTeams()
	if err != nil {
		return err
	}

	pctx.dbTeams = dbTeams

	// Load the posts directory
	posts, err := loadPosts(s.config.Posts)
	if err != nil {
		return err
	}

	pctx.posts = posts

	// Process all topics first
	err = s.processPostEntries("topic", &pctx)
	if err != nil {
		return err
	}

	// Delete removed posts
	err = s.cleanupRemovedPosts(pctx.dbTeamPosts)
	if err != nil {
		return err
	}

	// Refresh the list of posts
	pctx.dbTeamPosts, err = s.dbGetTeamPosts()
	if err != nil {
		return err
	}

	// Then the posts
	err = s.processPostEntries("post", &pctx)
	if err != nil {
		return err
	}

	// Then the posts
	err = s.processPostEntries("posts", &pctx)
	if err != nil {
		return err
	}

	return nil
}

func loadPosts(dir string) (map[string]post, error) {
	posts := map[string]post{}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(dir, file.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		newPost := post{}

		err = yaml.Unmarshal(content, &newPost)
		if err != nil {
			return nil, fmt.Errorf("failed to parse '%s': %w", path, err)
		}

		// Convert timestamps
		if newPost.Trigger != nil && newPost.Trigger.After != "" {
			ts, err := time.ParseInLocation("2006/01/02 15:04", newPost.Trigger.After, time.Local)
			if err != nil {
				return nil, err
			}

			newPost.Trigger.AfterTime = ts
		}

		name := strings.TrimSuffix(file.Name(), ".yaml")
		posts[name] = newPost
	}

	return posts, nil
}

func (s *syncer) cleanupRemovedPosts(dbTeamPosts map[int64]map[string][]int64) error {
	for _, entry := range dbTeamPosts {
		for name, postids := range entry {
			_, err := os.Lstat(filepath.Join(s.config.Posts, name+".yaml"))
			if err == nil || !os.IsNotExist(err) {
				continue
			}

			for _, postid := range postids {
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

	return nil
}

func selectTriggerTeams(p post, pctx *postContext) ([]dbTeam, error) {
	if p.Trigger == nil {
		return pctx.dbTeams, nil
	}

	teams := []dbTeam{}

	switch p.Trigger.Type {
	case "timer":
		if p.Trigger.AfterTime.Unix() > time.Now().Unix() {
			return teams, nil
		}

		return pctx.dbTeams, nil
	case "flag":
		for _, team := range pctx.dbTeams {
			if p.Trigger.Tag == "" {
				if pctx.askgodScores[team.AskgodID] == 0 {
					continue
				}
			} else if !int64InSlice(team.AskgodID, pctx.askgodFlags[p.Trigger.Tag]) {
				continue
			}

			teams = append(teams, team)
		}
	case "score":
		for _, team := range pctx.dbTeams {
			if pctx.askgodScores[team.AskgodID] < p.Trigger.Value {
				continue
			}

			teams = append(teams, team)
		}
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", p.Trigger.Type)
	}

	return teams, nil
}

func (s *syncer) processPostEntries(postType string, pctx *postContext) error {
	for name, p := range pctx.posts {
		if p.Type != postType {
			continue
		}

		// Sort out API keys
		apiUser := s.config.DiscourseAPIUser
		apiKey := s.config.DiscourseAPIKey

		if p.API != nil {
			apiUser = p.API.User
			apiKey = p.API.Key
		}

		teams, err := selectTriggerTeams(p, pctx)
		if err != nil {
			return err
		}

		err = s.publishPostToTeams(name, p, teams, apiUser, apiKey, pctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *syncer) publishPostToTeams(name string, p post, teams []dbTeam, apiUser string, apiKey string, pctx *postContext) error {
	for _, team := range teams {
		if len(s.config.PublishRestricted) > 0 && !stringInSlice(team.DiscourseName, s.config.PublishRestricted) {
			continue
		}

		_, ok := pctx.dbTeamPosts[team.AskgodID][name]
		if ok {
			// Already posted for this team, skip
			continue
		}

		if team.AskgodName == "" {
			team.AskgodName = team.DiscourseName
		}

		body := renderPostBody(p, team, pctx.askgodScores)

		err := s.dispatchPost(name, p, team, body, apiUser, apiKey, pctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderPostBody(p post, team dbTeam, askgodScores map[int64]int64) string {
	body := p.Body
	body = strings.ReplaceAll(body, "%{team_name}", team.AskgodName)
	body = strings.ReplaceAll(body, "%{team_score}", strconv.FormatInt(askgodScores[team.AskgodID], 10))

	r := regexp.MustCompile(`%\{(\w+)\}`)

	return r.ReplaceAllStringFunc(body, func(p2 string) string {
		return p.Variables[p2[2:len(p2)-1]][team.AskgodID]
	})
}

func (s *syncer) dispatchPost(name string, p post, team dbTeam, body string, apiUser string, apiKey string, pctx *postContext) error {
	switch p.Type {
	case "topic":
		return s.discourseCreateTopic(team.DiscourseName, team.AskgodID, apiUser, apiKey, name, team.DiscourseCategoryID, p.Title, body)
	case "post":
		for _, id := range pctx.dbTeamPosts[team.AskgodID][p.Topic] {
			err := s.discourseCreatePost(team.DiscourseName, team.AskgodID, apiUser, apiKey, name, id, body)
			if err != nil {
				return err
			}
		}

		return nil
	case "posts":
		return s.dispatchSubPosts(name, p, team, apiUser, apiKey, pctx)
	default:
		return fmt.Errorf("invalid type: %s", p.Type)
	}
}

func (s *syncer) dispatchSubPosts(name string, p post, team dbTeam, apiUser string, apiKey string, pctx *postContext) error {
	postIDs := pctx.dbTeamPosts[team.AskgodID][p.Topic]
	for _, subPost := range p.Posts {
		subAPIUser := apiUser
		subAPIKey := apiKey

		if subPost.API != nil {
			subAPIUser = subPost.API.User
			subAPIKey = subPost.API.Key
		}

		for _, id := range postIDs {
			err := s.discourseCreatePost(team.DiscourseName, team.AskgodID, subAPIUser, subAPIKey, name, id, subPost.Body)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
