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
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pixlise/core/v2/api/endpoints"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/utils"
	protos "github.com/pixlise/core/v2/generated-protos"
	"google.golang.org/protobuf/proto"
)

// Runs a quantification and checks that it steps through the expected states
func runQuantification(JWT string, environment string, datasetID string, pmcList []int32, elementList []string, detectorConfig string, quantName string) (string, error) {
	reqBody := quantModel.JobCreateParams{
		Name: quantName,
		//DatasetPath
		PMCs:           pmcList,
		Elements:       elementList,
		DetectorConfig: detectorConfig,
		Parameters:     "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
		RunTimeSec:     60,
		//RoiID
		//ElementSetID
		DatasetID: datasetID,
		//Creator
		QuantMode: "Combined",
		//RoiIDs
		//IncludeDwells
		Command: "map",
	}

	jsonBytes, err := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", generateURL(environment)+"/quantification/"+datasetID, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	bodyStr := string(body)

	jobID := ""
	if len(bodyStr) > 0 {
		// Get jobID from response body
		jobID = strings.TrimSpace(bodyStr)
		jobID = strings.ReplaceAll(jobID, `"`, "")
	}

	if resp.Status != "200 OK" {
		return jobID, fmt.Errorf("Error starting quantification: %v, response: %v", resp.Status, bodyStr)
	}

	// Now we wait, job should step through various states
	expectedJobStates := []string{"starting", "preparing_nodes", "nodes_running", "gathering_results", "complete"}

	// We allow a max quant run time of 5 minutes, should finish wayyy before then
	const maxRunTimeSec = 600
	const checkInterval = 15
	nextCheckInterval := 60 // we wait a bit longer for the first go, wasn't that reliable after 10sec.
	lastStatus := ""
	for c := 0; c < maxRunTimeSec/checkInterval; c++ {
		time.Sleep(time.Duration(nextCheckInterval) * time.Second)

		nextCheckInterval = checkInterval // subsequent checks are more frequent

		status, err := getQuantStatus(JWT, environment, datasetID, jobID)

		// If we ever fail to get a status back, stop here
		if err != nil {
			return jobID, fmt.Errorf("getQuantStatus failed for dataset %v: %v, response: %v", datasetID, resp.Status, err)
		}

		// Make sure the state returned is one of the ones we expect
		validStatus := false
		for _, expStatus := range expectedJobStates {
			if status == expStatus {
				validStatus = true
				break
			}
		}

		if !validStatus {
			return jobID, fmt.Errorf("Found unexpected job status '%v' for dataset %v, job id: %v", status, datasetID, jobID)
		}

		//if status != lastStatus {
		now := time.Now().Format(timeFormat)
		fmt.Printf(" %v    Job: %v, dataset: %v, status is: %v\n", now, jobID, datasetID, status)
		lastStatus = status
		//}

		if status == "complete" {
			break
		}
	}

	// If the status never completed in our wait time, that's an error
	if lastStatus != "complete" {
		return jobID, fmt.Errorf("Quant job: %v for dataset %v: timed out!", jobID, datasetID)
	}

	return jobID, err
}

func verifyQuantificationOKThenDelete(jobID string, JWT string, environment string, datasetID string, detectorConfig string, pmcList []int32, elementList []string, quantName string, exportColumns []string) error {
	resultPrint := printTestStart(fmt.Sprintf("Export of quantification: %v", jobID))
	exportColumnsStr := "["
	for c, col := range exportColumns {
		if c > 0 {
			exportColumnsStr += ","
		}
		exportColumnsStr += fmt.Sprintf("\"%v\"", col)
	}
	exportColumnsStr += "]"
	fileIds := []string{
		"raw-spectra",
		"quant-map-csv",
		"quant-map-tif",
		"beam-locations",
		"rois",
		"context-image",
		"unquantified-weight",
	}

	err := verifyExport(JWT, jobID, environment, datasetID, "export-test.zip", fileIds)
	printTestResult(err, resultPrint)
	if err != nil {
		return err
	}

	// Download the quant file
	resultPrint = printTestStart(fmt.Sprintf("Download and verify quantification: %v", jobID))
	// TODO ADD GENERATE URL
	quantBytes, err := checkFileDownload(JWT, generateURL(environment)+"/quantification/download/"+datasetID+"/"+jobID)

	if err == nil {
		// Downloaded, so check that we have the right # of PMCs and elements...
		err = checkQuantificationContents(quantBytes, pmcList, exportColumns)
	}
	printTestResult(err, resultPrint)
	if err != nil {
		return err
	}

	resultPrint = printTestStart(fmt.Sprintf("Delete generated quantification: %v for dataset: %v", jobID, datasetID))
	err = deleteQuant(JWT, jobID, environment, datasetID)
	printTestResult(err, resultPrint)

	return err
}

func checkQuantificationContents(quantBytes []byte, expPMCList []int32, expOutputElements []string) error {
	q := &protos.Quantification{}
	err := proto.Unmarshal(quantBytes, q)
	if err != nil {
		return err
	}

	// Verify the quant created as expected...
	if len(q.LocationSet) != 1 || q.LocationSet[0].Detector != "Combined" {
		return errors.New("Expected single detector named Combined")
	}

	// Make a lookup map for expected PMCs and output columns
	expPMCs := map[int32]bool{} // TODO: REFACTOR: Need generic utils.SetStringsInMap for this...
	for _, pmc := range expPMCList {
		expPMCs[pmc] = true
	}

	expElements := map[string]bool{}
	utils.SetStringsInMap(expOutputElements, expElements)

	keys := make([]int, 0, len(q.LocationSet[0].Location))

	for _, loc := range q.LocationSet[0].Location {
		pmc := loc.Pmc
		keys = append(keys, int(pmc))

		val, pmcExpected := expPMCs[pmc]
		if !pmcExpected {
			return fmt.Errorf("Quant contained unexpected PMC: %v", pmc)
		}
		if !val {
			return fmt.Errorf("Quant contained duplicated PMC: %v", pmc)
		}
		expPMCs[pmc] = false
	}

	sort.Ints(keys)

	// At the end, all our expected PMCs should've been found...
	for pmc, notFound := range expPMCs {
		if notFound {
			return fmt.Errorf("Quant missing expected PMC: %v", pmc)
		}
	}

	for _, label := range q.Labels {
		val, ok := expElements[label]
		if ok {
			// This is an expected label, ensure it's only found once and, mark it as found
			if !val {
				return fmt.Errorf("Quant contained duplicate column: %v", label)
			}
			expElements[label] = false
		}
	}

	for outputElem, notFound := range expElements {
		if notFound {
			return fmt.Errorf("Quant missing expected output element: %v", outputElem)
		}
	}

	return nil
}

func getQuantStatus(JWT string, environment string, datasetID string, jobID string) (string, error) {
	getReq, err := http.NewRequest("GET", generateURL(environment)+"/quantification/"+datasetID, nil)
	if err != nil {
		return "", err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return "", err
	}
	defer getResp.Body.Close()
	getBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return "", err
	}

	if getResp.Status != "200 OK" {
		return "", fmt.Errorf("Failed to get quant status for dataset %v: %v, response: %v", datasetID, getResp.Status, string(getBody))
	}

	var result endpoints.QuantListingResponse
	err = json.Unmarshal(getBody, &result)
	if err != nil {
		return "", err
	}

	// Finding where the current job is 7lq6mbw4sf2e8ehf
	jobIndex := -1
	for i := range result.Summaries {
		if result.Summaries[i].JobStatus.JobID == jobID {
			jobIndex = i
			break
		}
	}

	if jobIndex < 0 {
		return "", fmt.Errorf("Failed to find quant job: %v in quant list", jobID)
		//return "unknown", nil
	}

	return string(result.Summaries[jobIndex].JobStatus.Status), nil
}

// deletes quant after running it
func deleteQuant(JWT string, jobID string, environment string, datasetID string) error {
	req, err := http.NewRequest("DELETE", generateURL(environment)+"/quantification/"+datasetID+"/"+jobID, nil)
	if err != nil {
		return err
	}
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

	if resp.Status != "200 OK" {
		return fmt.Errorf("Failed to delete quantification: %v, response: %v", resp.Status, string(body))
	}
	return nil
}
