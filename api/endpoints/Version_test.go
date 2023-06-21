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

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/core/awsutil"
)

func Example_version() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/piquant-version.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/piquant-version.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"version": "registry.github.com/pixlise/piquant/runner:3.2.8",
	"changedUnixTimeSec": 1630635994,
	"creator": {
		"name": "Niko Belic",
		"user_id": "12345",
		"email": "nikobellic@gmail.com"
	}
}`))),
		},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil)
	apiRouter := MakeRouter(&svcs)

	req, _ := http.NewRequest("GET", "/", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(strings.HasPrefix(string(resp.Body.Bytes()), "<!DOCTYPE html>"))

	versionPat := regexp.MustCompile(`<h1>PIXLISE API</h1><p>Version .+</p>`)
	fmt.Println(versionPat.MatchString(string(resp.Body.Bytes())))

	req, _ = http.NewRequest("GET", "/version", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Printf("%v\n", resp.Body.String())

	req, _ = http.NewRequest("GET", "/version", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)

	if resp.Code != 200 {
		fmt.Printf("%v\n", resp.Body.String())
	}
	/*
		var ver ComponentVersionsGetResponse
		err := json.Unmarshal(resp.Body.Bytes(), &ver)
		fmt.Printf("%v\n", err)
		if err == nil {
			// Print out how many version structs and if we find an "API" one
			foundAPI := false
			foundPIQUANT := false
			for _, v := range ver.Components {
				if v.Component == "API" {
					foundAPI = true
				}
				if v.Component == "PIQUANT" {
					foundPIQUANT = true

					// Check it's what we expected
					fmt.Printf("PIQUANT version: %v\n", v.Version)
				}
			}

			vc := "not enough"
			if len(ver.Components) > 1 {
				vc = "ok"
			}
			fmt.Printf("Version count: %v, API found: %v, PIQUANT found: %v\n", vc, foundAPI, foundPIQUANT)
		}*/

	// Output:
	// 200
	// true
	// true
	// 500
	// PIQUANT version not set
	//
	// 200
	// <nil>
	// PIQUANT version: runner:3.2.8
	// Version count: ok, API found: true, PIQUANT found: true
}
