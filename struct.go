package main

import (
	"net/http"

	"gopkg.in/inconshreveable/log15.v2"
)

type syncer struct {
	config        *config
	logger        log15.Logger
	httpAskgod    *http.Client
	httpDiscourse *http.Client
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
