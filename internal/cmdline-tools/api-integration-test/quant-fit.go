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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pixlise/core/v2/core/quantModel"
)

// Runs a quantification and checks that it steps through the expected states
func runQuantFit(JWT string, environment string, datasetID string, pmcList []int32, elementList []string, detectorConfig string) error {
	reqBody := quantModel.JobCreateParams{
		//Name: quantName,
		//DatasetPath
		PMCs:           pmcList,
		Elements:       elementList,
		DetectorConfig: detectorConfig,
		Parameters:     "-Fe,1",
		RunTimeSec:     60,
		//RoiID
		//ElementSetID
		DatasetID: datasetID,
		//Creator
		QuantMode: "Combined",
		//RoiIDs
		//IncludeDwells
		Command: "quant",
	}

	jsonBytes, err := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", generateURL(environment)+"/quantification/"+datasetID, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyStr := string(body)

	if resp.Status != "200 OK" {
		return fmt.Errorf("Error running quant fit: %v, response: %v", resp.Status, bodyStr)
	}

	// Make sure we find the header for data from the FIT command
	idx := strings.Index(bodyStr, "Energy (keV), meas, calc, bkg, sigma, residual")
	if idx < 0 {
		return fmt.Errorf("Fit result did not contain expected column headers")
	}

	// Ensure that there is more than one \n after the index
	if strings.Count(bodyStr[idx+1:], "\\n") < 2 {
		return fmt.Errorf("Fit result did not contain data rows")
	}

	return nil
}
