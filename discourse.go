package main

import (
	"fmt"
	"strings"

	"gopkg.in/inconshreveable/log15.v2"
)

type discourseUser struct {
	ID                    int64  `json:"id"`
	Username              string `json:"username"`
	CanApprove            bool   `json:"can_approve"`
	RegistrationIPAddress string `json:"registration_ip_address"`
}

type discourseCategoryPost struct {
	Name        string            `json:"name"`
	Color       string            `json:"color"`
	TextColor   string            `json:"text_color"`
	Permissions map[string]string `json:"permissions"`
}

type discourseGroup struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type discourseGroupPost struct {
	Name         string `json:"name"`
	PrimaryGroup string `json:"primary_group"`
	Title        string `json:"title"`
}

// Users
func (s *syncer) discourseGetUsers() ([]discourseUser, error) {
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
	entry := group["basic_group"]

	return &entry, nil
}

func (s *syncer) discourseGroupExists(name string) bool {
	_, err := s.discourseGetGroup(name)
	if err != nil {
		return false
	}

	return true
}

func (s *syncer) discourseCreateGroup(name string) error {
	group := discourseGroupPost{
		Name:         name,
		Title:        fmt.Sprintf("Member of %s", name),
		PrimaryGroup: "true",
	}

	err := s.queryStruct("discourse", "POST", "/admin/groups/", group, nil)
	if err != nil {
		return err
	}

	return nil
}

// Categories
func (s *syncer) discourseCategoryExists(name string) bool {
	err := s.queryStruct("discourse", "GET", fmt.Sprintf("/c/%s.json", name), nil, nil)
	if err != nil {
		return false
	}

	return true
}

func (s *syncer) discourseCreateCategory(name string, groups []string) error {
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

	err := s.queryStruct("discourse", "POST", "/categories", category, nil)
	if err != nil {
		return err
	}

	return nil
}

// User setup
func (s *syncer) discourseSetupUser(user discourseUser, groups []string) error {
	// Setup the groups
	for _, group := range groups {
		adminGroup, err := s.discourseGetGroup(group)
		if err != nil {
			return fmt.Errorf("User group doesn't exist: %s", group)
		}

		// Add the user to the group
		member := map[string]string{
			"usernames": user.Username,
		}

		err = s.queryStruct("discourse", "PUT", fmt.Sprintf("/groups/%d/members", adminGroup.ID), member, nil)
		if err != nil {
			return err
		}
	}

	// Approve the user
	err := s.queryStruct("discourse", "PUT", fmt.Sprintf("/admin/users/%d/approve", user.ID), nil, nil)
	if err != nil {
		return err
	}

	// Activate the user (we don't do e-mails)
	err = s.queryStruct("discourse", "PUT", fmt.Sprintf("/admin/users/%d/activate", user.ID), nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *syncer) discourseProcessNewUsers() error {
	// Get all users
	users, err := s.discourseGetUsers()
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

		// Extract the discourse groups if any
		groups := strings.Split(team.Tags["discourse"], ";")

		// Activate the user
		err = s.discourseSetupUser(*adminUser, groups)
		if err != nil {
			s.logger.Error("Failed to setup new user", log15.Ctx{"user": adminUser.Username, "error": err})
			continue
		}

		s.logger.Info("Activated new user", log15.Ctx{"user": adminUser.Username})
	}

	return nil
}

// Team setup
func (s *syncer) discourseCreateTeams() error {
	// Get all the teams
	teams, err := s.askgodGetTeams()
	if err != nil {
		return err
	}

	for _, team := range teams {
		// Extract the discourse groups if any
		groups := strings.Split(team.Tags["discourse"], ";")

		// Create any needed groups and categories
		for _, group := range groups {
			if group == "" {
				continue
			}

			// Create the group if missing
			if !s.discourseGroupExists(group) {
				err = s.discourseCreateGroup(group)
				if err != nil {
					return err
				}

				s.logger.Info("Created new group", log15.Ctx{"name": group})
			}

			// Create the category if missing
			if strings.HasPrefix(group, s.config.CategoryFilter) && !s.discourseCategoryExists(group) {
				err = s.discourseCreateCategory(group, []string{group})
				if err != nil {
					return err
				}

				s.logger.Info("Created new category", log15.Ctx{"name": group})
			}
		}
	}

	return nil
}
