package main

import (
	"fmt"

	"github.com/pkg/errors"

	"gopkg.in/inconshreveable/log15.v2"
)

// Structs
type discourseUser struct {
	ID                    int64  `json:"id"`
	Username              string `json:"username"`
	CanApprove            bool   `json:"can_approve"`
	RegistrationIPAddress string `json:"registration_ip_address"`
}

type discourseCategoryPost struct {
	Name string `json:"name"`

	Color     string `json:"color"`
	TextColor string `json:"text_color"`

	Permissions map[string]string `json:"permissions"`
}

type discourseGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type discourseGroups struct {
	Groups []discourseGroup `json:"groups"`
}

type discourseGroupPost struct {
	Name         string `json:"name,omitempty"`
	FullName     string `json:"full_name,omitempty"`
	PrimaryGroup string `json:"primary_group,omitempty"`
	Title        string `json:"title,omitempty"`
}

// Users
func (s *syncer) discourseGetPendingUsers() ([]discourseUser, error) {
	users := []discourseUser{}

	err := s.queryStruct("discourse", "GET", "/admin/users/list/pending.json", nil, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *syncer) discourseGetUser(id int64) (*discourseUser, error) {
	user := discourseUser{}

	err := s.queryStruct("discourse", "GET", fmt.Sprintf("/admin/users/%d.json", id), nil, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Groups
func (s *syncer) discourseGetGroup(name string) (*discourseGroup, error) {
	// For some reason the response is wrapped
	group := map[string]discourseGroup{}

	err := s.queryStruct("discourse", "GET", fmt.Sprintf("/groups/%s.json", name), nil, &group)
	if err != nil {
		return nil, err
	}

	// Unwrap the response
	entry := group["group"]

	return &entry, nil
}

func (s *syncer) discourseCreateGroup(name string, fullName string) (int64, error) {
	title := ""
	if fullName == "" {
		fullName = name
		title = fmt.Sprintf("Member of %s", fullName)
	} else {
		title = fmt.Sprintf("Member of %s (%s)", fullName, name)
	}

	group := discourseGroupPost{
		Name:         name,
		FullName:     fullName,
		Title:        title,
		PrimaryGroup: "true",
	}

	var resp interface{}
	err := s.queryStruct("discourse", "POST", "/admin/groups/", group, &resp)
	if err != nil {
		return -1, err
	}

	return int64(resp.(map[string]interface{})["basic_group"].(map[string]interface{})["id"].(float64)), nil
}

func (s *syncer) discourseDeleteGroup(id int64) error {
	err := s.queryStruct("discourse", "DELETE", fmt.Sprintf("/admin/groups/%d", id), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *syncer) discourseUpdateGroup(id int64, name string, fullName string) error {
	title := ""
	if fullName == "" {
		fullName = name
		title = fmt.Sprintf("Member of %s", fullName)
	} else {
		title = fmt.Sprintf("Member of %s (%s)", fullName, name)
	}

	group := discourseGroupPost{
		FullName: fullName,
		Title:    title,
	}

	groupName := ""
	page := 0
	for {
		// The ID based API was removed, use the slow way
		groups := discourseGroups{}
		err := s.queryStruct("discourse", "GET", fmt.Sprintf("/groups.json?api_key=%s&api_username=%s&page=%d", s.config.DiscourseAPIKey, s.config.DiscourseAPIUser, page), nil, &groups)
		if err != nil {
			return err
		}

		if len(groups.Groups) == 0 {
			break
		}

		for _, group := range groups.Groups {
			if group.ID == id {
				groupName = group.Name
				break
			}
		}

		page += 1
	}

	if groupName == "" {
		return fmt.Errorf("Couldn't find group for id: %v", id)
	}

	err := s.queryStruct("discourse", "PUT", fmt.Sprintf("/groups/%v", groupName), group, nil)
	if err != nil {
		return err
	}

	return nil
}

// Categories
func (s *syncer) discourseCreateCategory(name string, groups []string) (int64, error) {
	category := discourseCategoryPost{
		Name:      name,
		Color:     s.config.CategoryColor,
		TextColor: s.config.CategoryTextColor,
	}

	permissions := map[string]string{}
	groups = append(groups, s.config.CategoryAccess...)
	for _, group := range groups {
		permissions[group] = "1"
	}
	category.Permissions = permissions

	var resp interface{}
	err := s.queryStruct("discourse", "POST", "/categories", category, &resp)
	if err != nil {
		return -1, err
	}

	return int64(resp.(map[string]interface{})["category"].(map[string]interface{})["id"].(float64)), nil
}

func (s *syncer) discourseDeleteCategory(id int64, name string) error {
	topics, err := s.discourseGetTopics(name)
	if err != nil {
		return err
	}

	for _, topic := range topics {
		s.discourseDeleteTopic(topic)
	}

	err = s.queryStruct("discourse", "DELETE", fmt.Sprintf("/categories/%d", id), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// Topics
func (s *syncer) discourseGetTopics(category string) ([]int64, error) {
	var resp interface{}
	err := s.queryStruct("discourse", "GET", fmt.Sprintf("/c/%s.json", category), nil, &resp)
	if err != nil {
		return nil, err
	}

	// Parse the response
	topics := []int64{}
	for _, entry := range resp.(map[string]interface{})["topic_list"].(map[string]interface{})["topics"].([]interface{}) {
		topics = append(topics, int64(entry.(map[string]interface{})["id"].(float64)))
	}

	return topics, nil
}

func (s *syncer) discourseCreateTopicAs(category int64, title string, body string, apiUser string, apiKey string) (int64, error) {
	post := map[string]interface{}{
		"category": category,
		"title":    title,
		"raw":      body,
	}

	if apiKey == "" {
		apiKey = s.config.DiscourseAPIKey
	}

	var resp interface{}
	err := s.queryStruct("discourse", "POST", fmt.Sprintf("/posts?api_username=%s&api_key=%s", apiUser, apiKey), post, &resp)
	if err != nil {
		return -1, err
	}

	return int64(resp.(map[string]interface{})["topic_id"].(float64)), nil
}

func (s *syncer) discourseDeleteTopic(id int64) error {
	err := s.queryStruct("discourse", "DELETE", fmt.Sprintf("/t/%d.json", id), nil, nil)
	if err != nil {
		return err
	}

	s.logger.Info("Deleted post", log15.Ctx{"id": id})
	return nil
}

// Posts
func (s *syncer) discourseCreatePostAs(topic int64, body string, apiUser string, apiKey string) (int64, error) {
	post := map[string]interface{}{
		"topic_id": topic,
		"raw":      body,
	}

	if apiKey == "" {
		apiKey = s.config.DiscourseAPIKey
	}

	var resp interface{}
	err := s.queryStruct("discourse", "POST", fmt.Sprintf("/posts?api_username=%s&api_key=%s", apiUser, apiKey), post, &resp)
	if err != nil {
		return -1, err
	}

	return int64(resp.(map[string]interface{})["id"].(float64)), nil
}

// User setup
func (s *syncer) discourseSetupUser(user discourseUser, group string) error {
	// Setup the groups
	adminGroup, err := s.discourseGetGroup(group)
	if err != nil {
		return fmt.Errorf("User group doesn't exist: %s", group)
	}

	// Add the user to the group
	member := map[string]string{
		"usernames": user.Username,
	}

	err = s.queryStruct("discourse", "PUT", fmt.Sprintf("/groups/%d/members.json", adminGroup.ID), member, nil)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to update groups for '%s'", user.Username))
	}

	// Approve the user
	err = s.queryStruct("discourse", "PUT", fmt.Sprintf("/admin/users/%d/approve", user.ID), nil, nil)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed to approve '%s'", user.Username))
	}

	return nil
}

func (s *syncer) discourseProcessNewUsers() error {
	// Get all users
	users, err := s.discourseGetPendingUsers()
	if err != nil {
		return err
	}

	for _, user := range users {
		// We only care about those that can be approved
		if !user.CanApprove {
			continue
		}

		// Get a full user record (including IP)
		adminUser, err := s.discourseGetUser(user.ID)
		if err != nil {
			s.logger.Error("Failed to get full user record", log15.Ctx{"user": user.Username, "error": err})
			continue
		}

		// Find what team they belong to
		team, err := s.askgodTeamForIP(adminUser.RegistrationIPAddress)
		if err != nil {
			s.logger.Error("Failed to find team for IP", log15.Ctx{"user": adminUser.Username, "ip": adminUser.RegistrationIPAddress, "error": err})
			continue
		}

		// Activate the user
		err = s.discourseSetupUser(*adminUser, team.Tags["discourse"])
		if err != nil {
			s.logger.Error("Failed to setup new user", log15.Ctx{"user": adminUser.Username, "error": err})
			continue
		}

		s.logger.Info("Activated new user", log15.Ctx{"user": adminUser.Username})
	}

	return nil
}

// Team setup
func (s *syncer) discourseCreateTeam(name string, id int64, title string) error {
	// Create the group
	groupID, err := s.discourseCreateGroup(name, title)
	if err != nil {
		return err
	}

	// Create the category
	categoryID, err := s.discourseCreateCategory(name, []string{name})
	if err != nil {
		return err
	}

	// Setup the DB entry
	err = s.dbCreateTeam(id, title, name, groupID, categoryID)
	if err != nil {
		return err
	}

	s.logger.Info("Created new team", log15.Ctx{"name": name, "title": title})
	return nil
}

func (s *syncer) discourseRenameTeam(name string, groupID int64, title string) error {
	// Set the fullName and title
	err := s.discourseUpdateGroup(groupID, name, title)
	if err != nil {
		return err
	}

	// Update the DB state
	err = s.dbRenameTeam(groupID, title)
	if err != nil {
		return err
	}

	s.logger.Info("Renamed team", log15.Ctx{"name": name, "title": title})
	return nil
}

func (s *syncer) discourseDeleteTeam(name string, groupID int64, categoryID int64) error {
	// Delete the category
	err := s.discourseDeleteCategory(categoryID, name)
	if err != nil {
		return err
	}

	// Delete the group
	err = s.discourseDeleteGroup(groupID)
	if err != nil {
		return err
	}

	// Setup the DB entry
	err = s.dbDeleteTeam(name, groupID, categoryID)
	if err != nil {
		return err
	}

	s.logger.Info("Deleted team", log15.Ctx{"name": name})
	return nil
}

func (s *syncer) discourseCreateTopic(name string, id int64, apiUser string, apiKey string, postName string, postCategory int64, postTitle string, postBody string) error {
	// Create the topic
	topicID, err := s.discourseCreateTopicAs(postCategory, postTitle, postBody, apiUser, apiKey)
	if err != nil {
		return err
	}

	// Setup the DB entry
	err = s.dbCreatePost(id, postName, topicID)
	if err != nil {
		return err
	}

	s.logger.Info("New topic", log15.Ctx{"team": name, "name": postName, "id": topicID})
	return nil

}

func (s *syncer) discourseCreatePost(name string, id int64, apiUser string, apiKey string, postName string, postID int64, postBody string) error {
	// Create the post
	postID, err := s.discourseCreatePostAs(postID, postBody, apiUser, apiKey)
	if err != nil {
		return err
	}

	// Setup the DB entry
	err = s.dbCreatePost(id, postName, postID)
	if err != nil {
		return err
	}

	s.logger.Info("New post", log15.Ctx{"team": name, "name": postName, "id": postID})
	return nil
}
