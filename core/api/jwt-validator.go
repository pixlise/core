// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

// api - package for containing "core" API things, which are reusable
// in building any API for our platform. These should not contain
// specific PIXLISE API business logic
package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pixlise/core/core/pixlUser"
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
