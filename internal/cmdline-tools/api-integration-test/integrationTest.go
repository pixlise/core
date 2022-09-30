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
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/pixlise/core/v2/core/auth0login"
	"github.com/pixlise/core/v2/core/utils"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	startupTime := time.Now()

	if len(os.Args) != 10 {
		fmt.Println("Arguments: environment, user, password, auth0_user_id, auth0_client_id auth0_secret auth0_domain auth0_audience expected_version")
		fmt.Println("  Where:")
		fmt.Println("  - environment name is one of [dev, staging, prod] OR a review environment name (eg review-env-blah, so without -api.review at the end)")
		fmt.Println("  - user - Auth0 user")
		fmt.Println("  - password - Auth0 password")
		fmt.Println("  - auth0_user_id - Auth0 user id (without Auth0| prefix)")
		fmt.Println("  - auth0_client_id - Auth0 API client id")
		fmt.Println("  - auth0_secret - Auth0 API secret")
		fmt.Println("  - auth0_domain - Auth0 API domain eg sometihng.au.auth0.com")
		fmt.Println("  - auth0_audience - Auth0 API audience")
		fmt.Println("  - expected_version is what we expect the API to return, eg 2.0.8-RC12. Or nil to skip check")
		os.Exit(1)
	}

	// Check arguments
	var environment = os.Args[1]

	fmt.Println("Running integration test for env: " + environment)

	var username = os.Args[2]
	var password = os.Args[3]
	var auth0UserID = os.Args[4]
	var auth0ClientID = os.Args[5]
	var auth0ClientSecret = os.Args[6]
	var auth0Domain = os.Args[7]
	var auth0Audience = os.Args[8]
	var expectedVersion = os.Args[9]

	// If expectedVersion is nil, clear it
	if expectedVersion == "nil" {
		expectedVersion = ""
	}

	printTestStart("API Version")
	err := checkAPIVersion(environment, expectedVersion)
	printTestResult(err, "")
	if err != nil {
		// If API version call is broken, probably everything is...
		os.Exit(1)
	}

	// TODO: Maybe we need to change this if we go open source?
	printTestStart("Getting JWT (Auth0 login)")
	JWT, err := auth0login.GetJWT(username, password, auth0ClientID, auth0ClientSecret, auth0Domain, "http://localhost:4200/authenticate", auth0Audience, "openid profile email")
	if err == nil && len(JWT) <= 0 {
		err = errors.New("JWT returned is empty")
	}
	printTestResult(err, "")
	if err != nil {
		// No point continuing, we couldn't log in!
		os.Exit(1)
	}

	// Check to see if there are alerts. If some come back we warn and check again, as maybe prev unit test run has left some over?
	printTestStart("Alerts (Before quantification tests)")
	preQuantAlerts, err := getAlerts(JWT, environment)
	if len(preQuantAlerts) > 0 {
		fmt.Printf(" WARNING: alerts came back with %v items. Will call again and verify it's cleared...\n", len(preQuantAlerts))
	}
	printTestResult(err, "")

	if len(preQuantAlerts) > 0 {
		// Re-check alerts, they should be empty now because the last call would've cleared them
		time.Sleep(3 * time.Second) // just in case...

		printTestStart("Alerts (Re-check)")
		alerts2, err2 := getAlerts(JWT, environment)
		if len(alerts2) > 0 {
			err2 = errors.New("Alerts expected to be empty after clearing")
		}
		printTestResult(err2, "")
	}

	printTestStart("Dataset listing")
	datasets, err := requestAndValidateDatasets(JWT, environment)
	printTestResult(err, "")
	if err != nil {
		os.Exit(1)
	}

	// Randomly pick a dataset and download its bin file and context image
	downloadTestIdx := rand.Int() % len(datasets)
	printTestStart(fmt.Sprintf("Downloading dataset binary file for: %v, id=%v", datasets[downloadTestIdx].Title, datasets[downloadTestIdx].DatasetID))
	_, err = checkFileDownload(JWT, datasets[downloadTestIdx].DataSetLink)
	printTestResult(err, "")

	if err == nil {
		printTestStart(fmt.Sprintf("Downloading dataset context image file for: %v, id=%v", datasets[downloadTestIdx].Title, datasets[downloadTestIdx].DatasetID))
		_, err = checkFileDownload(JWT, datasets[downloadTestIdx].ContextImageLink)
		printTestResult(err, "")
	}

	// Test quantifications on a few pre-determined datasets
	elementList := []string{"Ca", "Ti"}
	quantColumns := []string{"CaO_%", "TiO2_%"}
	detectorConfig := []string{"PIXL/v5", "PIXL/v5", "Breadboard/v1"}
	pmcsFor5x5 := []int32{}
	for c := 4043; c < 5806; c++ {
		if c != 4827 {
			pmcsFor5x5 = append(pmcsFor5x5, int32(c))
		}
	}
	pmcList := [][]int32{{68, 69, 70, 71}, pmcsFor5x5, {68, 69, 70, 71}}
	datasetIDs := []string{"983561", "test-fm-5x5-full", "test-kingscourt"} // test-laguna was timing out because saving the high rest TIFFs took longer than 1 minute, which seems to be the test limit

	// NOTE: By using 2 of the same names, we also test that the delete
	// didn't leave something behind and another can't be named that way
	quantNameSuffix := utils.RandStringBytesMaskImpr(8)
	quantNames := []string{"integration-test-same-name-" + quantNameSuffix, "integration-test-5x5-" + quantNameSuffix, "integration-test-same-name-" + quantNameSuffix}

	quantJobIDs := []string{}
	for i, datasetID := range datasetIDs {
		jobID := runQuantificationTestsForDataset(JWT, environment, datasetID, detectorConfig[i], pmcList[i], elementList, quantNames[i], quantColumns)
		if jobID == "" {
			printTestResult(fmt.Errorf("No JOB ID Returned for quant execution %v", quantNames[i]), "")
		}
		quantJobIDs = append(quantJobIDs, jobID)
	}

	// Test quant failing by supplying an invalid detector config (missing the /version)
	printTestStart("Check quantification failure return values")
	jobID, err := quantVerification(JWT, environment, "983561", pmcList[0], elementList, `"PIXL/v5"`, quantNames[0])
	if jobID != "" && (err != nil && err.Error() != "Error starting quantification: 400 Bad Request, response: DetectorConfig not in expected format") {
		printTestResult(fmt.Errorf("Unexpected result when running invalid quant: %v", err), "")
	} else {
		printTestResult(nil, "")
	}

	// Check that the expected alerts were generated during quantifications
	// This a start & finish alert for each job ID...
	expAlerts := map[string]bool{}

	for c, jobId := range quantJobIDs {
		qj := fmt.Sprintf("Started Quantification: %v (id: %v). Click on Quant Tracker tab to follow progress.", quantNames[c], jobId)
		qjf := fmt.Sprintf("Quantification %v Processing Complete", quantNames[c])
		fmt.Printf("%v\n", qj)
		fmt.Printf("%v\n", qjf)
		expAlerts[qj] = true
		expAlerts[qjf] = true
	}

	printTestStart("Alerts (Post quantification tests)")
	postQuantAlerts, err := getAlerts(JWT, environment)

	if err == nil {
		// NOTE: this covers the case where there are duplicate alerts coming in and we don't consider that an error!
		if len(postQuantAlerts) < len(expAlerts) {
			err = fmt.Errorf("Alerts came back with '%v' items, expected '%v'", len(postQuantAlerts), len(expAlerts))
		} else {
			if len(postQuantAlerts) > len(expAlerts)+1 {
				fmt.Printf(" WARNING: Got '%v' alerts, expected '%v'\n", len(postQuantAlerts), len(expAlerts))
			}

			// Check that they all match what we're expecting:
			// - Time range is anywhere from our test startup to now
			// - Text we've got in expAlerts
			// - User ID is known
			// - Topic we can deduce...
			currTime := time.Now()

			for _, alert := range postQuantAlerts {
				if alert.Timestamp.Before(startupTime) || alert.Timestamp.After(currTime) {
					err = fmt.Errorf("Alert timestamp was unexpected: %v", alert.Timestamp)
					break
				}

				if alert.UserID != auth0UserID {
					err = fmt.Errorf("Alert user ID was unexpected: %v", alert.UserID)
					break
				}

				if _, ok := expAlerts[alert.Message]; !ok {
					err = fmt.Errorf("Alert message was unexpected: %v. Available Messages:\n", alert.Message)
					for k, _ := range expAlerts {
						fmt.Printf("Message: %v\n", k)
					}
					break
				}

				// We should be able to work out the topic based on message
				expTopic := "Quantification Processing Start"
				if strings.HasSuffix(alert.Message, "Processing Complete") {
					expTopic = "Quantification Processing Complete"
					break
				}

				if alert.Topic != expTopic {
					err = fmt.Errorf("Alert topic was unexpected: %v", alert.Topic)
					break
				}
			}
		}
	}

	if err != nil {
		// Print out what was received, to aid debugging
		fmt.Printf("Alerts received: +%v\n", postQuantAlerts)
	}

	printTestResult(err, "")

	fmt.Println("\n==============================")
	/*
		if environment == "staging" || environment == "prod" {
			printTestStart("OCS Integration Test")
			err = runOCSTests()
			printTestResult(err, "")

			fmt.Println("\n==============================")

			printTestStart("Publish Integration Test")
			err = runPublishTests()
			printTestResult(err, "")

			fmt.Println("\n==============================")
		}
	*/

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
