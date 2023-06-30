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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pixlise/core/v3/core/pixlUser"
)

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

const timeFormat = "15:04:05" // "2006-01-02 15:04:05"
var lastStartedTestName = ""

func printTestStart(name string) string {
	timeNow := time.Now().Format(timeFormat)

	fmt.Println("---------------------------------------------------------")
	fmt.Printf(" %v TEST: %v\n", timeNow, name)
	//fmt.Println("---------------------------------------------------------")

	lastStartedTestName = name

	// Not even sure why this is returned anymore, seems it's not always passed as
	// name param to printTestResult, but we use lastStartedTestName now anyway
	return name
}

var failedTestNames = []string{}

func printTestResult(err error, name string) {
	suffix := ""
	if len(name) > 0 {
		suffix = " [" + name + "]"
	}

	timeNow := time.Now().Format(timeFormat)

	if err == nil {
		fmt.Printf(" %v  PASS%v", timeNow, suffix)
	} else {
		fmt.Printf(" %v  FAILED%v: %v\n", timeNow, suffix, err)
		failedTestNames = append(failedTestNames, lastStartedTestName)
	}
	fmt.Println("")
}

func getAlerts(JWT string, environment string) ([]pixlUser.UINotificationItem, error) {
	getReq, err := http.NewRequest("GET", generateURL(environment)+"/notification/alerts", nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()
	body, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	if getResp.Status != "200 OK" {
		return nil, fmt.Errorf("Alerts status fail: %v, response: %v", getResp.Status, string(body))
	}

	var alerts []pixlUser.UINotificationItem
	err = json.Unmarshal(body, &alerts)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}
