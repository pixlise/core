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

package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"

	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/permission"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"

	auth0 "github.com/auth0-community/go-auth0"
	jose "gopkg.in/square/go-jose.v2"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Authentication stuff

// See:
// https://auth0.com/blog/authentication-in-golang/
// https://github.com/auth0-community/auth0-go

const apiAudience = "pixlise-backend"

// Extracted from https://github.com/square/go-jose/blob/master/utils.go
// loadPublicKey loads a public key from PEM/DER-encoded data.
// You can download the Auth0 pem file from `applications -> your_app -> scroll down -> Advanced Settings -> certificates -> download`
func loadPublicKey(data []byte) (interface{}, error) {
	input := data

	block, _ := pem.Decode(data)
	if block != nil {
		input = block.Bytes
	}

	// Try to load SubjectPublicKeyInfo
	pub, err0 := x509.ParsePKIXPublicKey(input)
	if err0 == nil {
		return pub, nil
	}

	cert, err1 := x509.ParseCertificate(input)
	if err1 == nil {
		return cert.PublicKey, nil
	}

	return nil, fmt.Errorf("Public key parse error, got '%s' and '%s'", err0, err1)
}

type authMiddleWareData struct {
	routePermissionsRequired map[string]string
	jwtValidator             api.JWTInterface
	logger                   logger.ILogger
}

func isMatch(uri string, route string) bool {
	// Expect both to start with the same method
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	uriMethod := ""
	for c := range methods {
		if strings.HasPrefix(uri, methods[c]+"/") {
			uriMethod = methods[c]
			break
		}
	}

	// If we didn't find a method...
	if len(uriMethod) <= 0 {
		return false
	}

	// Make sure the route also had the same method
	if !strings.HasPrefix(route, uriMethod+"/") {
		return false
	}

	// See unit tests for what we match
	uriBits := strings.Split(strings.Trim(uri[len(uriMethod)+1:], "/"), "/")
	routeBits := strings.Split(strings.Trim(route[len(uriMethod)+1:], "/"), "/")

	// Must match in count
	if len(uriBits) != len(routeBits) {
		return false
	}

	// Match up until the {} start
	for c, uriBit := range uriBits {
		routeBit := routeBits[c]

		// If either is blank, something is wrong
		if len(uriBit) <= 0 || len(routeBit) <= 0 {
			return false
		}

		routeBitIsVar := len(routeBit) > 2 && routeBit[0:1] == "{" && routeBit[len(routeBit)-1:] == "}"

		if c > 0 && routeBitIsVar {
			// We don't check these, as it's a var replacement, but continue on in case the next element has to match...
			continue
		}

		if uriBit != routeBit {
			return false
		}
	}

	// Matched the above
	return true
}

func initJWTValidator(auth0Domain string, fs fileaccess.FileAccess, config config.APIConfig, apiLog logger.ILogger) *auth0.JWTValidator {
	// Create a configuration with the Auth0 information
	auth0PEM, err := fs.ReadObject(config.ConfigBucket, filepaths.GetConfigFilePath(filepaths.Auth0PemFileName))
	secret, err := loadPublicKey(auth0PEM)
	if err != nil {
		apiLog.Errorf("Couldn't load pem from S3: %v", err.Error())
	}

	secretProvider := auth0.NewKeyProvider(secret)
	audience := []string{apiAudience}

	configuration := auth0.NewConfiguration(secretProvider, audience, "https://"+auth0Domain+"/", jose.RS256)
	return auth0.NewValidator(configuration, nil)
}

func (a *authMiddleWareData) getPermissionsForURI(method string, uri string) (string, error) {
	// NOTE: we need to chop off query strings if any
	uriBits := strings.Split(uri, "?")
	if len(uriBits) > 1 {
		uri = uriBits[0]
	}
	// Try a direct match
	permissionRequired, ok := a.routePermissionsRequired[method+uri]
	if ok {
		return permissionRequired, nil
	}

	// No direct match, but we might find that it matches a URI that has {ids} in it
	for route, perm := range a.routePermissionsRequired {
		if isMatch(method+uri, route) {
			return perm, nil
		}
	}

	// No permission defined, so just fail it
	return "", fmt.Errorf("Permissions not defined for route: %v %v", method, uri)
}

func (a *authMiddleWareData) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the permission required for this route
		permissionRequired, err := a.getPermissionsForURI(r.Method, r.RequestURI)
		if err != nil {
			// No permission defined, so just fail it
			a.logger.Errorf("No permission found for URI %v. %v", r.RequestURI, err)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized - Bad route permissions"))
			return
		}

		// If we don't care about what permissions are required, it's public, so just allow it through
		if permissionRequired == permission.PermPublic {
			next.ServeHTTP(w, r)
			return
		}

		// Validate the token
		token, err := a.jwtValidator.ValidateRequest(r)

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized - Bad token"))
			return
		}

		// Read claims
		claims := map[string]interface{}{}
		err = a.jwtValidator.Claims(r, token, &claims)
		if err != nil {
			a.logger.Errorf("Failed to read claims from JWT: %v", err)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized - Bad claims"))
			return
		}

		// Make sure the permission required matches one of the claims
		permissions, err := api.ReadPermissions(claims)
		if err != nil {
			// No permission defined, so just fail it
			a.logger.Errorf("No permissions defined in claims. Error: %v", err)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized - Bad claim permissions"))
			return
		}

		// Check if it exists in permissions of user
		if !permissions[permissionRequired] {
			// Required permission is not in the claims of the JWT, so reject it
			a.logger.Errorf("Claim permissions did not contain %v for route: %v", permissionRequired, r.RequestURI)

			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized - Route not permitted"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
