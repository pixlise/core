package main

import (
	"flag"

	"github.com/pixlise/core/v3/api/config"
)

type IntegrationTestConfig struct {
	Environment       string // environment name is one of [dev, staging, prod] OR a review environment name (eg review-env-blah, so without -api.review at the end)
	Username          string // Auth0 user
	Password          string // Auth0 password
	Auth0UserID       string // Auth0 user id (without Auth0| prefix)
	Auth0ClientID     string // Auth0 API client id
	Auth0ClientSecret string // Auth0 API secret
	Auth0Domain       string // Auth0 API domain eg something.au.auth0.com
	Auth0Audience     string // Auth0 API audience
	ExpectedVersion   string // what we expect the API to return, eg 2.0.8-RC12. Or nil to skip check
}

func LoadConfig() (IntegrationTestConfig, error) {
	var cfg IntegrationTestConfig
	var err error

	configFilePath := flag.String("configPath", "", "path to the json file holding a set of custom config for the integration test")
	flag.Parse()

	if configFilePath != nil && *configFilePath != "" {
		cfg, err = config.NewConfigFromFile(*configFilePath, cfg)
	}
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}
