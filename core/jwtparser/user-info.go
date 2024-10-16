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

package jwtparser

import (
	"fmt"
	"net/http"
	"strings"
)

// IJWTReader - User ID getter from HTTP request
type IJWTReader interface {
	GetValidator() JWTInterface
	GetUserInfo(*http.Request) (JWTUserInfo, error)
}

type JWTUserInfo struct {
	Name        string          `json:"name"`
	UserID      string          `json:"user_id"`
	Email       string          `json:"email"`
	Permissions map[string]bool `json:"-" bson:"-"` // This is a lookup - we don't want this in JSON sent out of API though!
}

// RealJWTReader - Reader
type RealJWTReader struct {
	Validator JWTInterface
}

func (j RealJWTReader) GetValidator() JWTInterface {
	return j.Validator
}

// GetSimpleUserInfo - Get Simple User Info
// TODO: See note for GetUserInfo about user impersonation
func (j RealJWTReader) GetSimpleUserInfo(r *http.Request) (JWTUserInfo, error) {
	result := JWTUserInfo{}

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
// TODO: When a user is impersonating another, this still returns the real users details
//
//	from the JWT. This is unfortunate but seemed more effort than it's worth to fix
//	because we'll mainly test with science team members who have similar auth0
//	permissions, similar groups, similar access to datasets. If we do encounter
//	issues we can look at adding a map of userid->UserInfo structs containing the
//	impersonated users details, and return that from here
func (j RealJWTReader) GetUserInfo(r *http.Request) (JWTUserInfo, error) {
	result := JWTUserInfo{}

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

	// Also get permissions
	result.Permissions, err = ReadPermissions(claims)

	return result, err
}
