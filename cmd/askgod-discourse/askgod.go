package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/nsec/askgod/api"
	"gopkg.in/inconshreveable/log15.v2"
)

func (s *syncer) askgodGetTeams() ([]api.AdminTeam, error) {
	// Grab all the teams from askgod
	teams := []api.AdminTeam{}
	err := s.queryStruct("askgod", "GET", "/teams", nil, &teams)
	if err != nil {
		return nil, err
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
