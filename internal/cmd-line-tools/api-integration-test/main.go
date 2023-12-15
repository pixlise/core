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
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/wstestlib"
)

var auth0Params wstestlib.Auth0Info

var test1Username, test1Password, test2Username, test2Password string

// This is our complete integration test for the PIXLISE API
//
// It is intended to be run either locally on a dev laptop or in a test environment. It has several pre-requisites:
// - Running MongoDB where API is configured to talk to a database whose content we can wipe on integration test start
// - S3 buckets that API is configured to use, where we can wipe/replace files on integration test start
// - API, freshly started (so no cached things in memory yet)
// - 2 user accounts whose user/password is passed into here as arguments
//
// Integration test can then be started

var apiStorageFileAccess fileaccess.FileAccess
var apiDatasetBucket string
var apiUsersBucket string

func main() {
	rand.Seed(time.Now().UnixNano())
	//startupTime := time.Now()

	var apiHost string
	var apiDBSecret string
	var expectedAPIVersion string
	var testType string

	flag.StringVar(&apiHost, "apiHost", "", "Host name of API we're testing. Eg: localhost:8080 or something.review.pixlise.org")
	flag.StringVar(&apiDBSecret, "apiDBSecret", "", "Mongo secret of the DB the API is connected")
	flag.StringVar(&apiDatasetBucket, "datasetBucket", "", "Dataset bucket the API is using")
	flag.StringVar(&apiUsersBucket, "usersBucket", "", "User Data bucket the API is using")
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

	if strings.Contains(apiHost, "pixlise.org") {
		fmt.Printf("Can't connect to an env for now: %v\n", apiHost)
		return
	}

	fmt.Printf("Running integration test %v for %v\n", testType, apiHost)

	// Get a session for the bucket region
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS session. Error: %v", err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	apiStorageFileAccess = fileaccess.MakeS3Access(s3svc)

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

	// Connect to DB and drop the unit test database
	db := wstestlib.GetDB()
	err = db.Drop(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	dbCollections.InitCollections(db, &logger.StdOutLogger{})

	startTime := time.Now()
	runTests(apiHost)

	fmt.Println("\n==============================")

	if len(failedTestNames) == 0 {
		fmt.Printf("PASSED All Tests in %vsec!\n", time.Since(startTime).Seconds())
		os.Exit(0)
	}

	fmt.Printf("FAILED One or more tests at %vsec:\n", time.Since(startTime).Seconds())
	for _, name := range failedTestNames {
		fmt.Printf("- %v\n", name)
	}
	os.Exit(1)
}

func runTests(apiHost string) {
	testImageGet_PreWS(apiHost) // Must be run before any web sockets log in


	testImageUpload(apiHost)
	testImageMatchTransform(apiHost)
	testSelectionMsgs(apiHost)
	testQuants(apiHost)
	testDiffractionManualPeaks(apiHost)
	testDiffractionStatus(apiHost)
	testPiquantMsgs(apiHost)
	testExpressionRuntimeMsgs(apiHost)
	testDataModules(apiHost)
	testUserContent(apiHost, map[string]contentMessaging{
		"elementSet": {
			itemName: "elementSet",
			listName: "elementSets",
			invalidItemsToCreate: [][]string{
				{
					`"name": "User1 ElementSet1",
						"lines": []`,
					"Lines must contain at least 1 items",
				},
			},
			validItemsToCreate: [][]string{
				{
					`"name": "User1 ElementSet1",
						"lines": [
							{
								"Z":   14,
								"K":   false,
								"L":   false,
								"M":   true,
								"Esc": false
							},
							{
								"Z":   16,
								"K":   false,
								"L":   true,
								"M":   false,
								"Esc": true
							}
						]`,
					`"name": "User1 ElementSet1",
						"lines": [
							{
								"Z":   14,
								"M":   true
							},
							{
								"Z":   16,
								"L":   true,
								"Esc":   true
							}
						]`,
					`"name": "User1 ElementSet1",
						"atomicNumbers": [
							14, 16
						]`,
				},
			},
			invalidItemsToEdit: [][]string{
				{
					`"name": "This name is way way too long for any element set to seriously be named this way",
						"lines": [
							{
								"Z":   17,
								"K":   false,
								"L":   false,
								"M":   true,
								"Esc": false
							}
						]`,
					"Name is too long",
				},
			},
			validItemsToEdit: [][]string{
				{
					`"name": "User1 ElementSet1-Edited",
						"lines": [
							{
								"Z":   17,
								"K":   false,
								"L":   false,
								"M":   true,
								"Esc": false
							}
						]`,
					`"name": "User1 ElementSet1-Edited",
						"lines": [
							{
								"Z":   17,
								"M":   true
							}
						]`,
					`"name": "User1 ElementSet1-Edited",
						"atomicNumbers": [
							17
						]`,
				},
			},
			objectType: "OT_ELEMENT_SET",
		},
		"regionOfInterest": {
			itemName: "regionOfInterest",
			listName: "regionsOfInterest",
			invalidItemsToCreate: [][]string{
				{
					`"name": "Invalid ROI",
						"description": "Ye Invalid ROIe",
						"scanId": "048300551"`,
					"ROI must have location or pixel indexes defined",
				},
			},
			validItemsToCreate: [][]string{
				{
					`"name": "User1 ROI1",
						"description": "User1 ROI1",
						"scanId": "048300551",
						"scanEntryIndexesEncoded": [14, 123, -1, 126, 98]`,
					`"name": "User1 ROI1",
						"description": "User1 ROI1",
						"scanId": "048300551",
						"scanEntryIndexesEncoded": [14, 123, -1, 126, 98]`,
					`"name": "User1 ROI1",
						"description": "User1 ROI1",
						"scanId": "048300551"`,
				},
			},
			invalidItemsToEdit: [][]string{
				{
					`"scanId": "048300551",
						"name": "The ROI",
						"imageName": "WhatsAnImageDoingHere.png"`,
					"ROI image and pixel indexes must both be defined",
				},
			},
			validItemsToEdit: [][]string{
				{
					`"scanId": "048300551",
						"name": "The ROI",
						"description": "User1 ROI1",
						"scanEntryIndexesEncoded": [14, 123, -1, 126, 98, 88]`,
					`"scanId": "048300551",
						"name": "The ROI",
						"description": "User1 ROI1",
						"scanEntryIndexesEncoded": [14, 123, -1, 126, 98, 88]`,
					`"scanId": "048300551",
						"description": "User1 ROI1",
						"name": "The ROI"`,
				},
			},
			objectType: "OT_ROI",
		},
		"expressionGroup": {
			itemName: "group",
			listName: "groups",
			invalidItemsToCreate: [][]string{
				{
					`"name": "Invalid Expression Group",
						"groupItems": [{"expressionId": "abcd-123", "rangeMin": "12"}]`,
					"GroupItems is too short",
				},
			},
			validItemsToCreate: [][]string{
				{
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMin": 0, "rangeMax": 10},
							{"expressionId": "green-expr-1", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-1"}
						]`,
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMax": 10},
							{"expressionId": "green-expr-1", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-1"}
						]`,
					// List and item are the same
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMax": 10},
							{"expressionId": "green-expr-1", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-1"}
						]`,
				},
			},
			invalidItemsToEdit: [][]string{
				{
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix",
						"groupItems": [
							{"expressionId": "red-expr-2", "rangeMax": 10},
							{"expressionId": "green-expr-1", "rangeMin": 3, "rangeMax": 12},
							{"rangeMax": 133}]`,
					"[2].ExpressionId is too short",
				},
			},
			validItemsToEdit: [][]string{
				{
					`"description": "User1 RGBMix Modified",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMax": 19},
							{"expressionId": "green-expr-3", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-6"}
						]`,
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix Modified",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMax": 19},
							{"expressionId": "green-expr-3", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-6"}
						]`,
					// List and item are the same
					`"name": "User1 ExpressionGroup",
						"description": "User1 RGBMix Modified",
						"groupItems": [
							{"expressionId": "red-expr-1", "rangeMax": 19},
							{"expressionId": "green-expr-3", "rangeMin": 4, "rangeMax": 12},
							{"expressionId": "blue-expr-6"}
						]`,
				},
			},
			objectType: "OT_EXPRESSION_GROUP",
		},
		"expression": {
			itemName: "expression",
			listName: "expressions",
			invalidItemsToCreate: [][]string{
				{
					`"name": "Invalid Expression",
					"sourceLanguage": "LUA"`,
					"SourceCode is too short",
				},
			},
			validItemsToCreate: [][]string{
				{
					`"name": "User1 Expression",
					"comments": "User1 Expression",
					"sourceLanguage": "LUA",
					"sourceCode": "element(\"Ca\")"`,
					`"name": "User1 Expression",
					"sourceLanguage": "LUA",
					"comments": "User1 Expression",
					"sourceCode": "element(\"Ca\")"`,
					`"name": "User1 Expression",
					"sourceLanguage": "LUA",
					"comments": "User1 Expression"`,
				},
			},
			invalidItemsToEdit: [][]string{
				{
					`"name": "User1 Expression Invalid",
					"comments": "User1 Expression Invalid",
					"sourceLanguage": "PYTHON"`,
					"Invalid source language: PYTHON",
				},
			},
			validItemsToEdit: [][]string{
				{
					`"comments": "User1 Expression Modified",
					"sourceCode": "element(\"Fe\")"`,
					`"name": "User1 Expression",
					"comments": "User1 Expression Modified",
					"sourceLanguage": "LUA",
					"sourceCode": "element(\"Fe\")"`,
					`"name": "User1 Expression",
					"comments": "User1 Expression Modified",
					"sourceLanguage": "LUA"`,
				},
			},
			objectType: "OT_EXPRESSION",
		},
	})

	testUserSearch(apiHost)
	testUserDetails(apiHost)
	testUserManagement(apiHost)
	testUserGroups(apiHost)
	testLogMsgs(apiHost)
	testScanData(apiHost, 0 /*3 for proper testing*/)
	testDetectorConfig(apiHost)

	testTags(apiHost)
	testROIUserConfiguration(apiHost)
	testScreenConfiguration(apiHost)
	testWidgetData(apiHost)
	testDOI(apiHost)
}
