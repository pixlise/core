// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Contains all the code needed to do an Auth0 login and retrieve a JWT.
// This is useful for command line tools that access the API and for integration tests
// It is not intended to be used by the API runtime itself, that should only ever
// parse JWTs passed to it in HTTP requests, it shouldn't generate an Auth0 JWT for
// any internal reasons
package auth0login

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pixlise/core/v4/api/config"
	"gopkg.in/auth0.v4/management"
)

// Auth0TokenResponse - The token response type
type Auth0TokenResponse struct {
	AccessToken string `json:"access_token"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// GetJWT performs an Auth0 login using the parameters and returns the JWT if successful
func GetJWT(username string, password string, clientID string, clientSecret string, auth0domain string, redirectURI string, audience string, scope string) (string, error) {
	// Form the request
	reqURL := fmt.Sprintf("https://%v/oauth/token", auth0domain)
	payload := strings.NewReader(fmt.Sprintf("grant_type=password&username=%v&password=%v&audience=%v&scope=%v&client_id=%v&client_secret=%v", username, password, audience, scope, clientID, clientSecret))
	resp, err := http.Post(reqURL, "application/x-www-form-urlencoded", payload)
	if err != nil {
		return "", fmt.Errorf("Auth0 login failed: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read login response: %v", err)
	}

	bodyData := Auth0TokenResponse{}
	err = json.Unmarshal(body, &bodyData)
	if err != nil {
		return "", fmt.Errorf("Failed to parse login response: %v", err)
	}

	if len(bodyData.AccessToken) <= 0 {
		return "", fmt.Errorf("Failed to get access token: %v", string(body))
	}

	return bodyData.AccessToken, nil
}

func InitAuth0ManagementAPI(cfg config.APIConfig) (*management.Management, error) {
	api, err := management.New(cfg.Auth0Domain, cfg.Auth0ManagementClientID, cfg.Auth0ManagementSecret)
	return api, err
}
