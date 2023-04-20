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

// api - package for containing "core" API things, which are reusable
// in building any API for our platform. These should not contain
// specific PIXLISE API business logic
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pixlise/core/v3/core/pixlUser"
	"gopkg.in/square/go-jose.v2/jwt"
)

type JWTInterface interface {
	ValidateRequest(r *http.Request) (*jwt.JSONWebToken, error)
	Claims(r *http.Request, token *jwt.JSONWebToken, values ...interface{}) error
}

// RealJWTReader - Reader
type RealJWTReader struct {
	Validator JWTInterface
}

// GetSimpleUserInfo - Get Simple User Info
func (j RealJWTReader) GetSimpleUserInfo(r *http.Request) (pixlUser.UserInfo, error) {
	result := pixlUser.UserInfo{}

	// Get user ID
	token, err := j.Validator.ValidateRequest(r)
	if err != nil {
		return result, err
	}

	// Read claims
	claims := map[string]interface{}{}
	err = j.Validator.Claims(r, token, &claims)
	if err != nil {
		return result, err
	}

	userNameObj, ok := claims["https://pixlise.org/username"]
	if !ok {
		return result, fmt.Errorf("Failed to get user name from request JWT")
	}
	result.Name = userNameObj.(string)

	userIDObj, ok := claims["sub"]
	if !ok {
		return result, fmt.Errorf("Failed to get user ID from request JWT")
	}

	result.UserID = userIDObj.(string)
	pipePos := strings.Index(result.UserID, "|")
	if pipePos > -1 {
		result.UserID = result.UserID[pipePos+1:]
	}

	return result, err
}

// GetUserInfo - Get User Info
func (j RealJWTReader) GetUserInfo(r *http.Request) (pixlUser.UserInfo, error) {
	result := pixlUser.UserInfo{}

	// Get user ID
	token, err := j.Validator.ValidateRequest(r)
	if err != nil {
		return result, err
	}

	// Read claims
	claims := map[string]interface{}{}
	err = j.Validator.Claims(r, token, &claims)
	if err != nil {
		return result, err
	}

	userNameObj, ok := claims["https://pixlise.org/username"]
	if !ok {
		return result, fmt.Errorf("Failed to get user name from request JWT")
	}
	result.Name = userNameObj.(string)

	userEmailObj, ok := claims["https://pixlise.org/email"]
	if !ok {
		return result, fmt.Errorf("Failed to get email address from JWT")
	}

	result.Email = userEmailObj.(string)

	userIDObj, ok := claims["sub"]
	if !ok {
		return result, fmt.Errorf("Failed to get user ID from request JWT")
	}

	result.UserID = userIDObj.(string)
	pipePos := strings.Index(result.UserID, "|")
	if pipePos > -1 {
		result.UserID = result.UserID[pipePos+1:]
	}

	// Also get permissions
	result.Permissions, err = ReadPermissions(claims)

	return result, err
}

func ReadPermissions(claims map[string]interface{}) (map[string]bool, error) {
	result := map[string]bool{}

	claimPermissions, ok := claims["permissions"].([]interface{}) // example of casting interface to something concrete
	if !ok {
		return result, fmt.Errorf("Failed to get permissions from request JWT")
	}

	for _, claimPerm := range claimPermissions {
		// Get it as a string
		claimPermStr, ok := claimPerm.(string)
		if ok {
			result[claimPermStr] = true
		}
	}

	return result, nil
}
