// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package endpoints

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
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
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

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
	}

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
