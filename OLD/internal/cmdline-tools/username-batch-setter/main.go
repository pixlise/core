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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/pixlise/core/v3/api/endpoints"
	"github.com/pixlise/core/v3/core/auth0login"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/utils"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 9 {
		fmt.Println("Arguments: environment, user, password, auth0_client_id auth0_secret auth0_domain auth0_audience filename")
		fmt.Println("  Where:")
		fmt.Println("  - environment name is one of [dev, staging, prod] OR a review environment name (eg review-env-blah, so without -api.review at the end)")
		fmt.Println("  - user - Auth0 user")
		fmt.Println("  - password - Auth0 password")
		fmt.Println("  - auth0_client_id - Auth0 API client id")
		fmt.Println("  - auth0_secret - Auth0 API secret")
		fmt.Println("  - auth0_domain - Auth0 API domain eg something.au.auth0.com")
		fmt.Println("  - auth0_audience - Auth0 API audience")
		fmt.Println("  - filename - Name of JSON file containing user batch changes")
		os.Exit(1)
	}

	// Check arguments
	var environment = os.Args[1]

	fmt.Println("Running username batch setter for env: " + environment)

	var username = os.Args[2]
	var password = os.Args[3]
	var auth0ClientID = os.Args[4]
	var auth0ClientSecret = os.Args[5]
	var auth0Domain = os.Args[6]
	var auth0Audience = os.Args[7]
	var batchFileName = os.Args[8]

	JWT, err := auth0login.GetJWT(username, password, auth0ClientID, auth0ClientSecret, auth0Domain, "http://localhost:4200/authenticate", auth0Audience, "openid profile email")
	if err == nil && len(JWT) <= 0 {
		err = errors.New("JWT returned is empty")
	}

	if err != nil {
		log.Fatalf("%v\n", err)
	}

	localFS := fileaccess.FSAccess{}

	userEdits := []endpoints.UserEditRequest{}
	err = localFS.ReadJSON(batchFileName, "", &userEdits, false)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	if len(userEdits) < 1 {
		log.Fatalf("Expected at least one user edit item\n")
	}

	// Upload
	userEditData, err := json.MarshalIndent(userEdits, "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	req, err := http.NewRequest("POST", generateURL(environment)+"/user/bulk-user-details", bytes.NewReader(userEditData))
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	req.Header.Set("Authorization", "Bearer "+JWT)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	fmt.Printf("Result: %v [%v]. Body: %v\n", resp.StatusCode, resp.Status, string(body))
}

// Copied from integration test! Should probably move this to common code...
func generateURL(environment string) string {
	url := "https://"
	// Prod or other fixed environments...
	if environment == "prod" {
		url += "www-api"
	} else if environment == "dev" || environment == "staging" || environment == "test" {
		url += environment + "-api"
	} else {
		// Review environments have more stuff added on, eg:
		// https://review-env-blah-api.review.pixlise.org
		url += environment + "-api.review"
	}

	url += ".pixlise.org"
	return url
}
