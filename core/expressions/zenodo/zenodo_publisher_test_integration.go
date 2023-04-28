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

package zenodo

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/pixlise/core/v3/core/expressions/modules"
	"github.com/pixlise/core/v3/core/pixlUser"
)

func Test_create_empty_deposition(t *testing.T) {
	t.Setenv("ZENODO_URI", "https://sandbox.zenodo.org")
	t.Setenv("ZENODO_ACCESS_TOKEN", "TEST_TOKEN")

	accessToken := os.Getenv("ZENODO_ACCESS_TOKEN")
	if accessToken == "" {
		t.Errorf("Failed to find ZENODO_ACCESS_TOKEN")
	}

	zenodoURI := os.Getenv("ZENODO_URI")
	if zenodoURI == "" {
		t.Errorf("Failed to find ZENODO_URI")
	}

	deposition, err := createEmptyDeposition(zenodoURI, accessToken)
	if err != nil {
		t.Errorf("Failed to create empty deposition: %v", err)
	}

	if deposition == nil {
		t.Errorf("Deposition is nil")
	} else if deposition.Links.Bucket == "" {
		t.Errorf("Deposition.Links.Bucket is empty")
	} else if deposition.Links.LatestDraft == "" {
		t.Errorf("Deposition.Links.LatestDraft is empty")
	} else if deposition.Links.Publish == "" {
		t.Errorf("Deposition.Links.Publish is empty")
	}
}

func Test_upload_file_to_deposition(t *testing.T) {
	t.Setenv("ZENODO_URI", "https://sandbox.zenodo.org")
	t.Setenv("ZENODO_ACCESS_TOKEN", "TEST_TOKEN")

	accessToken, foundAccessToken := os.LookupEnv("ZENODO_ACCESS_TOKEN")
	if !foundAccessToken {
		t.Errorf("Failed to find ZENODO_ACCESS_TOKEN")
	}

	zenodoURI, foundZenodoURI := os.LookupEnv("ZENODO_URI")
	if !foundZenodoURI {
		t.Errorf("Failed to find ZENODO_URI")
	}

	deposition, err := createEmptyDeposition(zenodoURI, accessToken)
	if err != nil {
		t.Errorf("Failed to create empty deposition: %v", err)
	}

	data := map[string]string{
		"data": "this is a test",
	}

	filename := "test.json"
	jsonContents, err := json.Marshal(data)
	if err != nil {
		t.Errorf("Failed to marshal test data: %v", err)
	}

	fileUploadResponse, err := uploadFileContentsToZenodo(*deposition, filename, bytes.NewBuffer([]byte(jsonContents)), accessToken)
	if err != nil {
		t.Errorf("Failed to upload test file contents to Zenodo: %v", err)
	}

	if fileUploadResponse == nil {
		t.Errorf("File upload response is nil")
	} else if fileUploadResponse.Key == "" {
		t.Errorf("File upload response.Key is empty, probably malformed data")
	}

	testModule := modules.DataModuleSpecificVersionWire{
		DataModule: &modules.DataModule{ID: "mod123",
			Name:     "TestModule",
			Comments: "This is a test",
			Origin: pixlUser.APIObjectItem{
				Shared:              true,
				Creator:             pixlUser.UserInfo{Name: "Ryan S", UserID: "333", Email: "ryan@pixlise.org"},
				CreatedUnixTimeSec:  1234567889,
				ModifiedUnixTimeSec: 1234567892,
			},
		},
		Version: modules.DataModuleVersionSourceWire{
			SourceCode: "element(\"Ca\", \"%\", \"A\")",
			DataModuleVersionWire: &modules.DataModuleVersionWire{
				Version:          "2.1.43",
				Tags:             []string{"latest", "experimental", "test"},
				Comments:         "This is a test",
				TimeStampUnixSec: 1234567891,
			},
		},
	}

	fileUploadResponse, err = uploadModuleToZenodo(*deposition, testModule, accessToken)
	if err != nil {
		t.Errorf("Failed to upload test module to Zenodo: %v", err)
	}

	if fileUploadResponse == nil {
		t.Errorf("File upload response is nil for test module")
	} else if fileUploadResponse.Key == "" {
		t.Errorf("File upload response.Key is empty for test module, probably malformed file data")
	}

	metadataResponse, err := addModuleMetadataToDeposition(*deposition, testModule, accessToken)
	if err != nil {
		t.Errorf("Failed to add metadata to deposition: %v", err)
	}

	if metadataResponse == nil {
		t.Errorf("Metadata response is nil")
	} else if metadataResponse.ConceptRecID == "" {
		t.Errorf("Metadata response.ConceptRecID is empty, probably malformed metadata")
	}

	publishResponse, err := publishDeposition(*deposition, accessToken)
	if err != nil {
		t.Errorf("Failed to publish deposition: %v", err)
	}

	if publishResponse == nil {
		t.Errorf("Publish response is nil")
	} else if publishResponse.Submitted != true {
		t.Errorf("Publish response.Submitted is false, test data was not published!")
	}
}
