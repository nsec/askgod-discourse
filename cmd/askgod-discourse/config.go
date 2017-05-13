package main

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	AskgodURL  string `yaml:"askgod_url"`
	AskgodCert string `yaml:"askgod_cert"`

	Database string `yaml:"database"`

	DiscourseURL     string `yaml:"discourse_url"`
	DiscourseCert    string `yaml:"discourse_cert"`
	DiscourseAPIKey  string `yaml:"discourse_api_key"`
	DiscourseAPIUser string `yaml:"discourse_api_user"`

	CategoryAccess    []string `yaml:"category_access"`
	CategoryColor     string   `yaml:"category_color"`
	CategoryTextColor string   `yaml:"category_text_color"`
}

func parseConfig(path string) (*config, error) {
	// Read the file's content
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file content: %v", err)
	}

	// Parse the yaml file
	config := config{}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse yaml: %v", err)
	}

	return &config, nil
}
