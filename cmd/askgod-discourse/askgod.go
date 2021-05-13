package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/nsec/askgod/api"
)

func (s *syncer) askgodGetTeams() ([]api.AdminTeam, error) {
	// Grab all the teams from askgod
	teams := []api.AdminTeam{}
	err := s.queryStruct("askgod", "GET", "/teams", nil, &teams, nil)
	if err != nil {
		return nil, err
	}

	return teams, nil
}

func (s *syncer) askgodGetTeamDiscourseFlags() (map[string][]int64, error) {
	// Get all the flags
	flags := []api.AdminFlag{}
	err := s.queryStruct("askgod", "GET", "/flags", nil, &flags, nil)
	if err != nil {
		return nil, err
	}

	// Get all the scores
	scores := []api.AdminScore{}
	err = s.queryStruct("askgod", "GET", "/scores", nil, &scores, nil)
	if err != nil {
		return nil, err
	}

	// Generate the output
	resp := map[string][]int64{}

	for _, flag := range flags {
		if flag.Tags["discourse"] == "" {
			continue
		}

		teams := []int64{}
		for _, score := range scores {
			if score.FlagID == flag.ID {
				teams = append(teams, score.TeamID)
			}
		}

		resp[flag.Tags["discourse"]] = teams
	}

	return resp, nil
}

func (s *syncer) askgodGetTeamScores() (map[int64]int64, error) {
	// Grab the scoreboard
	board := []api.ScoreboardEntry{}
	err := s.queryStruct("askgod", "GET", "/scoreboard", nil, &board, nil)
	if err != nil {
		return nil, err
	}

	teams := map[int64]int64{}
	for _, entry := range board {
		teams[entry.Team.ID] = entry.Value
	}

	return teams, nil
}

func (s *syncer) askgodTeamForIP(ipStr string) (*api.AdminTeam, error) {
	// Parse the IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("Bad IP: %s", ipStr)
	}

	// Get all the teams
	teams, err := s.askgodGetTeams()
	if err != nil {
		return nil, err
	}

	// Iterate for one that matches the IP
	for _, team := range teams {
		if team.Subnets == "" {
			continue
		}

		// Teams can have multiple subnets
		subnets := strings.Split(team.Subnets, ",")
		for _, subnet := range subnets {
			subnet = strings.TrimSpace(subnet)

			// Parse the subnet
			_, netSubnet, err := net.ParseCIDR(subnet)
			if err != nil {
				s.logger.Error("Bad team subnet", log15.Ctx{"error": err, "subnet": subnet})
				continue
			}

			// Check if the IP is in the subnet
			if netSubnet.Contains(ip) {
				return &team, nil
			}
		}
	}

	return nil, fmt.Errorf("No team matches the subnet")
}
