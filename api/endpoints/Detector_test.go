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
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
