package main

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

type config struct {
	AskgodURL  string `yaml:"askgod_url"`
	AskgodCert string `yaml:"askgod_cert"`

	Database string `yaml:"database"`
	Posts    string `yaml:"posts"`

	DiscourseURL     string `yaml:"discourse_url"`
	DiscourseCert    string `yaml:"discourse_cert"`
	DiscourseAPIKey  string `yaml:"discourse_api_key"`
	DiscourseAPIUser string `yaml:"discourse_api_user"`

	CategoryAccess    []string `yaml:"category_access"`
	CategoryColor     string   `yaml:"category_color"`
	CategoryTextColor string   `yaml:"category_text_color"`
	CategoryParent    string   `yaml:"category_parent"`

	PublishRestricted []string `yaml:"publish_restricted"`
}

func parseConfig(path string) (*config, error) {
	// Read the file's content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Parse the yaml file
	config := config{}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	return &config, nil
}
