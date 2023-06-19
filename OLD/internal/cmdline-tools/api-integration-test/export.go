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
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

// checks the exporter works on the hardcoded datasets added below
func verifyExport(JWT string, jobID string, environment string, datasetID string, fileName string, fileIds []string) error {
	jsonStr := `{"fileName": "` + fileName + `", "quantificationId":"` + jobID + `", "fileIds":[`
	for c, file := range fileIds {
		if c > 0 {
			jsonStr += ","
		}
		jsonStr += "\"" + file + "\""
	}

	jsonStr += `]}`
	var jsonBytes = []byte(jsonStr)
	req, err := http.NewRequest("POST", generateURL(environment)+"/export/files/"+datasetID, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+JWT)

	client := &http.Client{
		Timeout: 0, //time.Second * 300,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read zip body: %v", err)
	}

	if resp.Status != "200 OK" {
		return fmt.Errorf("Export status fail: %v, response: %v", resp.Status, string(body))
	}

	// Check response headers
	expContentDisposition := `attachment; filename="` + fileName + `"`
	if resp.Header["Content-Disposition"][0] != expContentDisposition {
		return fmt.Errorf("Missing Content-Disposition from response header")
	}

	if resp.Header["Content-Length"][0] == "0" {
		return fmt.Errorf("Unexpected content length")
	}

	// Check body zip contents
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("Failed to read zip content: %v", err)
	}

	// Check that the files seem remotely correct...
	fileNamePrefix := fileName[0 : len(fileName)-4]
	expectedFileNames := map[string]bool{
		fileNamePrefix + "-map-by-PIQUANT.csv":                true,
		fileNamePrefix + "-beam-locations.csv":                true,
		fileNamePrefix + "-unquantified-weight-pct.csv":       true,
		fileNamePrefix + "-Normal-BulkSum ROI All Points.csv": true,
		fileNamePrefix + "-Normal ROI All Points.csv":         true,
		//fileNamePrefix + "-roi-pmcs.csv":                      true, // not including this check as files are generated per ROI now
		//fileNamePrefix + "-Dwell-BulkSum ROI All Points.csv":  true,
		//fileNamePrefix + "-Dwell ROI All Points.csv":          true,
	}
	/*hasPNG hasTIF, hasTXT,*/ hasCSV := false //, false, false, false

	for _, zipFile := range zipReader.File {
		/*if !hasPNG && strings.HasSuffix(zipFile.Name, ".png") {
			hasPNG = true
		}
		if !hasTIF && strings.HasSuffix(zipFile.Name, ".tif") {
			hasTIF = true
		}
		if !hasTXT && strings.HasSuffix(zipFile.Name, ".txt") {
			hasTXT = true
		}*/
		if !hasCSV && strings.HasSuffix(zipFile.Name, ".csv") {
			hasCSV = true
		}

		_, ok := expectedFileNames[zipFile.Name]
		if ok {
			expectedFileNames[zipFile.Name] = false
		}
	}

	// If didn't see any files of these types...
	if /*hasPNG == false || hasTIF == false || hasTXT == false ||*/ hasCSV == false {
		return fmt.Errorf("One or more files missing from export zip")
	}

	// If didn't see any of the expected file names...
	for k, v := range expectedFileNames {
		if v {
			return fmt.Errorf("Export zip did not contain: %v", k)
		}
	}

	return err
}
