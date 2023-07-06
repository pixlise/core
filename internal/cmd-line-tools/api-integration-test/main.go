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
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/pixlise/core/v3/core/wstestlib"
)

var auth0Params wstestlib.Auth0Info

var test1Username, test1Password, test2Username, test2Password string

func main() {
	rand.Seed(time.Now().UnixNano())
	//startupTime := time.Now()

	var apiHost string
	var apiDBSecret string
	var expectedAPIVersion string
	var testType string

	flag.StringVar(&apiHost, "apiHost", "", "Host name of API we're testing. Eg: localhost:8080 or something.review.pixlise.org")
	flag.StringVar(&apiDBSecret, "apiDBSecret", "", "Mongo secret of the DB the API is connected")
	flag.StringVar(&auth0Params.Domain, "auth0Domain", "", "Auth0 domain for management API")
	flag.StringVar(&auth0Params.ClientId, "auth0ClientId", "", "Auth0 client id for management API")
	flag.StringVar(&auth0Params.Secret, "auth0Secret", "", "Auth0 secret for management API")
	flag.StringVar(&auth0Params.Audience, "auth0Audience", "", "Auth0 audience")
	flag.StringVar(&expectedAPIVersion, "expectedAPIVersion", "", "Expected API version (version not checked if blank)")
	flag.StringVar(&testType, "testType", "endpoints", "Test type to run: endpoints, short")

	flag.StringVar(&test1Username, "test1Username", "", "Username of test account 1")
	flag.StringVar(&test1Password, "test1Password", "", "Password of test account 1")
	flag.StringVar(&test2Username, "test2Username", "", "Username of test account 2")
	flag.StringVar(&test2Password, "test2Password", "", "Password of test account 2")

	flag.Parse()

	fmt.Printf("Running integration test %v for %v\n", testType, apiHost)

	if len(expectedAPIVersion) > 0 {
		printTestStart("API Version")
		err := checkAPIVersion(apiHost, expectedAPIVersion)
		printTestResult(err, "")
		if err != nil {
			// If API version call is broken, probably everything is...
			os.Exit(1)
		}
	}

	if testType != "endpoints" {
		log.Fatal("Unexpected test type: " + testType)
	}

	runTests(apiHost)

	fmt.Println("\n==============================")

	if len(failedTestNames) == 0 {
		fmt.Println("PASSED All Tests!")
		os.Exit(0)
	}

	fmt.Println("FAILED One or more tests:")
	for _, name := range failedTestNames {
		fmt.Printf("- %v\n", name)
	}
	os.Exit(1)
}

func runTests(apiHost string) {
	testUserDetails(apiHost)
	testElementSets(apiHost)
	testUserManagement(apiHost)
}
