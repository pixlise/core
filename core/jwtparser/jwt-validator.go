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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/auth0-community/go-auth0"
	"github.com/pixlise/core/v3/core/fileaccess"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

const apiAudience = "pixlise-backend"

// Implements a JWT validation and claim extraction interface
type JWTInterface interface {
	ValidateRequest(r *http.Request) (*jwt.JSONWebToken, error)
	Claims(r *http.Request, token *jwt.JSONWebToken, values ...interface{}) error
}

func InitJWTValidator(auth0Domain string, configBucket string, pemPath string, fs fileaccess.FileAccess) (*auth0.JWTValidator, error) {
	// Create a configuration with the Auth0 information
	auth0PEM, err := fs.ReadObject(configBucket, pemPath)
	secret, err := loadPublicKey(auth0PEM)
	if err != nil {
		return nil, fmt.Errorf("Failed to load PEM file: %v", err.Error())
	}

	secretProvider := auth0.NewKeyProvider(secret)
	audience := []string{apiAudience}

	configuration := auth0.NewConfiguration(secretProvider, audience, "https://"+auth0Domain+"/", jose.RS256)

	return auth0.NewValidator(configuration, nil), nil

	// NOTE: if we have to extract the token from multiple places we can do this...
	/*auth0.FromMultiple(
		auth0.RequestTokenExtractorFunc(auth0.FromHeader),
		auth0.RequestTokenExtractorFunc(auth0.FromParams),
	))*/
}

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
