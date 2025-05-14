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
	"flag"
	"fmt"

	"github.com/pixlise/core/v4/core/auth0login"
)

// This test program implements the Auth0 "resource-owner-password" flow documented here:
// https://auth0.com/docs/api/authentication#resource-owner-password
// NOTE: this was a pain to get working, the primary problem being when configuring the
// account in the Auth0 web interface, ticking the "Password" "Grant type" and hitting
// Save caused an error saying "Client credentials" is not allowed. Even though we didn't
// tick it, and it was disabled. Turns out it's a UI bug, and I had to set our application
// type to Native, change "Token Endpoint Authentication Method" to not be "None", then
// disable "Client credentials", hit Save, set application back to "Single page app",
// and finally I was able to tick "Password" and save.
//
// A similar complication was looming, as the login was failing due to there not being
// a default database set for users. Instead of doing it the Auth0 settings web interface
// way, I decided to use password-realm as the grant type, where we can specify the realm
// here.
//
//

func main() {
	var username string
	var password string
	var clientID string
	var auth0domain string
	var redirectURI string
	var audience string
	var scope string

	flag.StringVar(&username, "user", "", "Username")
	flag.StringVar(&password, "pass", "", "Password")
	flag.StringVar(&clientID, "id", "", "Client ID")

	flag.StringVar(&auth0domain, "domain", "pixlise.au.auth0.com", "Auth0 Domain (optional)")
	flag.StringVar(&audience, "audience", "pixlise-backend", "Auth0 Audience (optional)")
	flag.StringVar(&scope, "scope", "openid profile email", "Auth0 Scope (optional)")
	flag.StringVar(&redirectURI, "redirecturi", "http://localhost:4200/authenticate", "Auth0 Redirect URI (optional)")

	flag.Parse()

	jwt, err := auth0login.GetJWT(username, password, clientID, auth0domain, redirectURI, audience, scope)
	fmt.Printf("%v|%v\n", err, jwt)
}
