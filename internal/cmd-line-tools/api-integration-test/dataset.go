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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	datasetModel "github.com/pixlise/core/v3/core/dataset"
)

// Requests all dataset summaries, and verifies they're well formatted
func requestAndValidateDatasets(JWT string, environment string) ([]datasetModel.APIDatasetSummary, error) {
	var result = []datasetModel.APIDatasetSummary{}
	req, err := http.NewRequest("GET", generateURL(environment)+"/dataset", nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", "Bearer "+JWT)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return result, err
	}

	//storing errors in a dynamic slice to allow for multiple errors across multiple datasets
	//to be outputted to better diagnosis issues
	errCount := 0
	downloadLimit := 1
	for c, item := range result {
		dserror := isValidDatasetItem(item, JWT)

		if dserror != nil {
			fmt.Printf("%v\n", dserror)
			errCount++
		}

		if c >= downloadLimit {
			break
		}
	}

	if errCount > 0 {
		return result, errors.New("Dataset query failed")
	}

	fmt.Printf(" Received %v dataset summaries\n", len(result))
	return result, nil
}

// Ensures dataset summary has valid fields and has no errors. Returns an error, or nil if no error
func isValidDatasetItem(dataset datasetModel.APIDatasetSummary, JWT string) error {
	datasetIDPat := regexp.MustCompile(`.+`)
	if !datasetIDPat.MatchString(dataset.DatasetID) {
		return errors.New("Missing DatasetID")
	}

	if len(dataset.ContextImage) == 0 {
		// Context image count should be 0
		// For now, that 1 dataset we have without a context image set generates with count of 1 so allow this
		if dataset.ContextImages != 0 && dataset.ContextImages != 1 {
			return fmt.Errorf("Expected 0 Context Images in Dataset: %v", dataset.DatasetID)
		}
	} else {
		// Validate the image is a file name, and context image count > 0
		contextImagePat := regexp.MustCompile(`.+(.jpg|.png)`)
		if len(dataset.ContextImage) > 0 && contextImagePat.MatchString(dataset.ContextImage) == false {
			return fmt.Errorf("Missing Context Image in Dataset: %v", dataset.DatasetID)
		}

		if dataset.ContextImages == 0 {
			return fmt.Errorf("Expected > 0 Context Images in Dataset: %v", dataset.DatasetID)
		}
	}

	if dataset.DataFileSize == 0 {
		return fmt.Errorf("Invalid Data File Size in Dataset: %v", dataset.DatasetID)
	}

	if len(dataset.DetectorConfig) <= 0 {
		return fmt.Errorf("Missing Detector Config in Dataset: %v", dataset.DatasetID)
	}

	datasetLinkPat := regexp.MustCompile(`https://.+/dataset`)
	datasetLink := datasetLinkPat.FindString(dataset.DataSetLink)
	if datasetLink == "" {
		return fmt.Errorf("Missing Dataset Image Link in Dataset: %v", dataset.DatasetID)
	}

	// If no context image, expect no link... otherwise expect both
	if len(dataset.ContextImage) > 0 {
		// Expect link to be set correctly too
		contextImageLinkPat := regexp.MustCompile(`https://.*(.png|.jpg)`)
		contextImageLink := contextImageLinkPat.FindString(dataset.ContextImageLink)
		if len(contextImageLink) <= 0 {
			return fmt.Errorf("Missing Context Image Link in Dataset: %v", dataset.DatasetID)
		}
	} else {
		// Context image is empty, ensure link is too
		if len(dataset.ContextImageLink) > 0 {
			return fmt.Errorf("Context image is empty, expected link to be empty also, got: %v", dataset.ContextImageLink)
		}
	}

	return nil
}
