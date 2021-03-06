package main

import (
	"database/sql"
	"net/http"
	"sync"

	"github.com/inconshreveable/log15"
)

type syncer struct {
	config        *config
	logger        log15.Logger
	httpAskgod    *http.Client
	httpDiscourse *http.Client
	db            *sql.DB

	postsLock sync.Mutex
	teamsLock sync.Mutex
}

func getSyncer(path string) (*syncer, error) {
	s := syncer{}

	// Setup logging
	s.logger = log15.New()

	// Setup config
	config, err := parseConfig(path)
	if err != nil {
		return nil, err
	}

	s.config = config

	// Setup askgod client
	client, err := s.getClient(config.AskgodURL, config.AskgodCert)
	if err != nil {
		return nil, err
	}

	s.httpAskgod = client

	// Setup discourse client
	client, err = s.getClient(config.DiscourseURL, config.DiscourseCert)
	if err != nil {
		return nil, err
	}

	s.httpDiscourse = client

	return &s, nil
}
