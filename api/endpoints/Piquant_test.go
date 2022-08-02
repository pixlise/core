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
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/awsutil"
)

func Example_detectorQuantConfigHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{Bucket: aws.String(ConfigBucketForUnitTest), Prefix: aws.String("DetectorConfig/")},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("DetectorConfig/PetersSuperDetector/pixlise-config.json")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/V1/config.json")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/2.0/optic.txt")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/2.0/config.json")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/V1/optic.txt")},
				{Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/v2.1-broken/file.txt")},
				{Key: aws.String("DetectorConfig/AnotherConfig/pixlise-config.json")},
				{Key: aws.String("DetectorConfig/AnotherConfig/PiquantConfigs/V1/config.json")},
			},
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/piquant/config", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Returns config names in alphabetical order

	// Output:
	// 200
	// {
	//     "configNames": [
	//         "AnotherConfig",
	//         "PetersSuperDetector"
	//     ]
	// }
}

func Example_detectorQuantConfigHandler_VersionList() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

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

	req, _ := http.NewRequest("GET", "/piquant/config/PetersSuperDetector/versions", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// [
	//     "V1",
	//     "2.0"
	// ]
}

func Example_detectorQuantConfigHandler_GetNotFound() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("DetectorConfig/WeirdDetector/pixlise-config.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/piquant/config/WeirdDetector/version/v1.1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// WeirdDetector not found
}

func Example_detectorQuantConfigHandler_GetOK() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("DetectorConfig/PetersSuperDetector/pixlise-config.json"),
		},
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("DetectorConfig/PetersSuperDetector/PiquantConfigs/ver1.1/config.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"minElement": 11,
	"maxElement": 92,
	"xrfeVLowerBound": 800,
	"xrfeVUpperBound": 20000,
	"xrfeVResolution": 230,
	"windowElement": 4,
	"tubeElement": 14,
	"somethingUnknown": 42,
	"defaultParams": "-q,xyzPIELKGHTXCRNFSVM7ijsrdpetoubaln -b,0,8,50 -f",
	"mmBeamRadius": 0.007
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"description": "Peters super detector config",
	"config-file": "config.msa",
	"optic-efficiency": "optical.csv",
	"calibration-file": "calibration.csv",
	"standards-file": "standards.csv"
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/piquant/config/PetersSuperDetector/version/ver1.1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "pixliseConfig": {
	//         "minElement": 11,
	//         "maxElement": 92,
	//         "xrfeVLowerBound": 800,
	//         "xrfeVUpperBound": 20000,
	//         "xrfeVResolution": 230,
	//         "windowElement": 4,
	//         "tubeElement": 14,
	//         "defaultParams": "-q,xyzPIELKGHTXCRNFSVM7ijsrdpetoubaln -b,0,8,50 -f",
	//         "mmBeamRadius": 0.007
	//     },
	//     "quantConfig": {
	//         "description": "Peters super detector config",
	//         "configFile": "config.msa",
	//         "opticEfficiencyFile": "optical.csv",
	//         "calibrationFile": "calibration.csv",
	//         "standardsFile": "standards.csv"
	//     }
	// }
}

func Example_detectorQuantConfigHandler_OtherMethods() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/piquant/config/WeirdDetector/version/1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("PUT", "/piquant/config/WeirdDetector/version/1", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("DELETE", "/piquant/config/WeirdDetector/version/1", nil)
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

func Example_piquantDownloadHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	const artifactBucket = "our-artifact-bucket"

	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(artifactBucket), Prefix: aws.String("piquant/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("piquant/invalid-path.txt"), LastModified: aws.Time(time.Unix(1597124080, 0)), Size: aws.Int64(12)},
				{Key: aws.String("piquant/piquant-linux-2.7.1.zip"), LastModified: aws.Time(time.Unix(1597124080, 0)), Size: aws.Int64(1234)},
				{Key: aws.String("piquant/piquant-windows-2.6.0.zip"), LastModified: aws.Time(time.Unix(1597124000, 0)), Size: aws.Int64(12345)},
			},
		},
	}

	var mockSigner awsutil.MockSigner
	mockSigner.Urls = []string{"http://signed-url.com/file1.zip", "http://signed-url.com/file2.zip"}

	svcs := MakeMockSvcs(&mockS3, nil, &mockSigner, nil, nil)
	svcs.Config.BuildsBucket = artifactBucket

	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/piquant/download", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "downloadItems": [
	//         {
	//             "buildVersion": "2.7.1",
	//             "buildDateUnixSec": 1597124080,
	//             "fileName": "piquant-linux-2.7.1.zip",
	//             "fileSizeBytes": 1234,
	//             "downloadUrl": "http://signed-url.com/file1.zip",
	//             "os": "linux"
	//         },
	//         {
	//             "buildVersion": "2.6.0",
	//             "buildDateUnixSec": 1597124000,
	//             "fileName": "piquant-windows-2.6.0.zip",
	//             "fileSizeBytes": 12345,
	//             "downloadUrl": "http://signed-url.com/file2.zip",
	//             "os": "windows"
	//         }
	//     ]
	// }
}

func Example_piquantDownloadHandler_OtherMethods() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/piquant/download", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("PUT", "/piquant/download", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("DELETE", "/piquant/download", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/piquant/download/some-id", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
	//
	// 405
	//
	// 405
	//
	// 404
	// 404 page not found
}

func Example_piquantHandler_GetVersion() {
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
    "version": "registry.github.com/pixlise/piquant/runner:3.0.7-ALPHA",
    "changedUnixTimeSec": 1234567890,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.Config.PiquantDockerImage = "registry.github.com/pixlise/piquant/runner:3.0.8-ALPHA"
	apiRouter := MakeRouter(svcs)

	// Success, we have config var set, returns that
	req, _ := http.NewRequest("GET", "/piquant/version", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Success, S3 overrides config var
	req, _ = http.NewRequest("GET", "/piquant/version", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	//
	// 200
	// {
	//     "version": "registry.github.com/pixlise/piquant/runner:3.0.8-ALPHA",
	//     "changedUnixTimeSec": 0,
	//     "creator": {
	//         "name": "",
	//         "user_id": "",
	//         "email": ""
	//     }
	// }
	//
	// 200
	// {
	//     "version": "registry.github.com/pixlise/piquant/runner:3.0.7-ALPHA",
	//     "changedUnixTimeSec": 1234567890,
	//     "creator": {
	//         "name": "Niko Bellic",
	//         "user_id": "600f2a0806b6c70071d3d174",
	//         "email": "niko@spicule.co.uk"
	//     }
	// }
}

func Example_piquantHandler_GetVersion_NoConfigVar() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/piquant-version.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Fails, no S3 and no config var
	req, _ := http.NewRequest("GET", "/piquant/version", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// PIQUANT version not found
}

func Example_piquantHandler_SetVersion() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	//fmt.Println(string(expBinBytes))
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(ConfigBucketForUnitTest), Key: aws.String("PixliseConfig/piquant-version.json"), Body: bytes.NewReader([]byte(`{
    "version": "registry.github.com/pixlise/piquant/runner:3.0.7-ALPHA",
    "changedUnixTimeSec": 1234567777,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567777},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/piquant/version", bytes.NewReader([]byte(`{
    "version": "registry.github.com/pixlise/piquant/runner:3.0.7-ALPHA"
}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
}
