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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/core/awsutil"
)

func Example_detectorConfigHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("DetectorConfig/WeirdDetector/pixlise-config.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("DetectorConfig/PetersSuperDetector/pixlise-config.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"minElement": 11,
	"maxElement": 92,
	"xrfeVLowerBound": 800,
	"xrfeVUpperBound": 20000,
	"xrfeVResolution": 230,
	"windowElement": 4,
	"tubeElement": 14,
	"mmBeamRadius": 0.3,
	"somethingUnknown": 42,
	"defaultParams": "-q,xyzPIELKGHTXCRNFSVM7ijsrdpetoubaln -b,0,8,50 -f"
}`))),
		},
	}

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(ConfigBucketForUnitTest), Prefix: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/V1/config.json")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/2.0/optic.txt")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/2.0/config.json")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/V1/optic.txt")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/v2.1-broken/file.txt")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/detector-config/WeirdDetector", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/detector-config/PetersSuperDetector", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// WeirdDetector not found
	//
	// 200
	// {
	//     "minElement": 11,
	//     "maxElement": 92,
	//     "xrfeVLowerBound": 800,
	//     "xrfeVUpperBound": 20000,
	//     "xrfeVResolution": 230,
	//     "windowElement": 4,
	//     "tubeElement": 14,
	//     "defaultParams": "-q,xyzPIELKGHTXCRNFSVM7ijsrdpetoubaln -b,0,8,50 -f",
	//     "mmBeamRadius": 0.3,
	//     "piquantConfigVersions": [
	//         "V1",
	//         "2.0"
	//     ]
	// }
}

func Example_detectorConfigHandler_OtherMethods() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/detector-config/WeirdDetector", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("PUT", "/detector-config/WeirdDetector", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("DELETE", "/detector-config/WeirdDetector", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
	//
	// 405
	//
	// 405
}
