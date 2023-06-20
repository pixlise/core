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

// GetSimpleUserInfo - Get Simple User Info
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
	pipePos := strings.Index(result.UserID, "|")
	if pipePos > -1 {
		result.UserID = result.UserID[pipePos+1:]
	}

	// Also get permissions
	result.Permissions, err = ReadPermissions(claims)

	return result, err
}
