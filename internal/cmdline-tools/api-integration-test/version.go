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
	"strings"

	"github.com/pixlise/core/v2/api/endpoints"
)

// Checks API version is valid (just as a string, checks with regex, does not check against an expected deployed version!)
func checkAPIVersion(environment string, expectedVersion string) error {
	resp, err := http.Get(generateURL(environment) + "/version")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result endpoints.ComponentVersionsGetResponse
	err = json.Unmarshal(body, &result)
	//fmt.Printf("%v", string(body))

	if err != nil {
		return err
	}

	theApiVersion := ""

	for _, ver := range result.Components {
		if ver.Component == "API" {
			theApiVersion = ver.Version
			break
		}
	}

	// If we don't have an expected version....
	if len(expectedVersion) <= 0 {
		// Just make sure it's not blank
		if len(theApiVersion) <= 0 {
			return fmt.Errorf("Error fetching API version, got: %v", theApiVersion)
		}

		fmt.Printf("  API version returned: %v\n", theApiVersion)
	} else {
		// NOTE: we assume user has provided the start of the version, but the API now appends the git hash too
		// so this check works if user only supplies 1.2.3 or 1.2.3-RC1 OR the entire thing 1.2.3-RC1-githash
		if !strings.HasPrefix(theApiVersion, expectedVersion) {
			return fmt.Errorf("Expected API version \"%v\", got: \"%v\"", expectedVersion, theApiVersion)
		}

		fmt.Printf("  API version matched: %v\n", theApiVersion)
	}

	return nil
}
