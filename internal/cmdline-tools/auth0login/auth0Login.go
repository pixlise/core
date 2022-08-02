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

package main

import (
	"flag"
	"fmt"

	"github.com/pixlise/core/core/auth0login"
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
	var clientSecret string
	var auth0domain string
	var redirectURI string
	var audience string
	var scope string

	flag.StringVar(&username, "user", "", "Username")
	flag.StringVar(&password, "pass", "", "Password")
	flag.StringVar(&clientID, "id", "", "Client ID")
	flag.StringVar(&clientSecret, "secret", "", "Client Secret")

	flag.StringVar(&auth0domain, "domain", "pixlise.au.auth0.com", "Auth0 Domain (optional)")
	flag.StringVar(&audience, "audience", "pixlise-backend", "Auth0 Audience (optional)")
	flag.StringVar(&scope, "scope", "openid profile email", "Auth0 Scope (optional)")
	flag.StringVar(&redirectURI, "redirecturi", "http://localhost:4200/authenticate", "Auth0 Redirect URI (optional)")

	flag.Parse()

	jwt, err := auth0login.GetJWT(username, password, clientID, clientSecret, auth0domain, redirectURI, audience, scope)
	fmt.Printf("%v|%v\n", err, jwt)
}
