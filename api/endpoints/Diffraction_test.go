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
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
)

const statusS3Path = "UserContent/shared/rtt-123/diffraction-peak-statuses.json"
const manualS3Path = "UserContent/shared/rtt-123/manual-diffraction-peaks.json"
const statusFile = `{
	"id-1": "not-anomaly",
	"id-2": "intensity-mismatch"
}`
const manualFile = `{
	"peaks": {
		"id-1": {
			"pmc": 32,
			"keV": 5.6
		},
		"id-2": {
			"pmc": 44,
			"keV": 7.7
		}
	}
}`

func Example_diffractionHandler_ListAccepted() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(statusFile))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/diffraction/status/rtt-123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/diffraction/status/rtt-123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/diffraction/status/rtt-123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {}
	//
	// 500
	// invalid character 'g' looking for beginning of value
	//
	// 200
	// {
	//     "id-1": "not-anomaly",
	//     "id-2": "intensity-mismatch"
	// }
}

func Example_diffractionHandler_PostStatuses() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(statusFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(statusFile))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path), Body: bytes.NewReader([]byte(`{
    "new-1": "diffraction"
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path), Body: bytes.NewReader([]byte(`{
    "new-2": "other"
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path), Body: bytes.NewReader([]byte(`{
    "id-1": "not-anomaly",
    "id-2": "intensity-mismatch",
    "new-3": "weird-one"
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path), Body: bytes.NewReader([]byte(`{
    "id-1": "not-anomaly",
    "id-2": "not-anomaly"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File missing, first go, should just create
	req, _ := http.NewRequest("POST", "/diffraction/status/diffraction/rtt-123/new-1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Should ignore the fact that the incoming file is garbage, and write a new one
	req, _ = http.NewRequest("POST", "/diffraction/status/other/rtt-123/new-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// New appended to existing list
	req, _ = http.NewRequest("POST", "/diffraction/status/weird-one/rtt-123/new-3", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Checking no duplicates made
	req, _ = http.NewRequest("POST", "/diffraction/status/not-anomaly/rtt-123/id-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "new-1": "diffraction"
	// }
	//
	// 200
	// {
	//     "new-2": "other"
	// }
	//
	// 200
	// {
	//     "id-1": "not-anomaly",
	//     "id-2": "intensity-mismatch",
	//     "new-3": "weird-one"
	// }
	//
	// 200
	// {
	//     "id-1": "not-anomaly",
	//     "id-2": "not-anomaly"
	// }
}

func Example_diffractionHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(statusFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(statusFile))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(statusS3Path), Body: bytes.NewReader([]byte(`{
    "id-1": "not-anomaly"
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File missing, 404
	req, _ := http.NewRequest("DELETE", "/diffraction/status/rtt-123/new-1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Incoming is garbage, 500
	req, _ = http.NewRequest("DELETE", "/diffraction/status/rtt-123/new-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Not found in list
	req, _ = http.NewRequest("DELETE", "/diffraction/status/rtt-123/new-3", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// OK
	req, _ = http.NewRequest("DELETE", "/diffraction/status/rtt-123/id-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// new-1 not found
	//
	// 500
	// invalid character 'g' looking for beginning of value
	//
	// 404
	// new-3 not found
	//
	// 200
	// {
	//     "id-1": "not-anomaly"
	// }
}

func Example_diffractionHandler_ListManual() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(manualFile))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/diffraction/manual/rtt-123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/diffraction/manual/rtt-123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/diffraction/manual/rtt-123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {}
	//
	// 500
	// invalid character 'g' looking for beginning of value
	//
	// 200
	// {
	//     "id-1": {
	//         "pmc": 32,
	//         "keV": 5.6
	//     },
	//     "id-2": {
	//         "pmc": 44,
	//         "keV": 7.7
	//     }
	// }
}

func Example_diffractionHandler_PostManual() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(manualFile))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path), Body: bytes.NewReader([]byte(`{
    "peaks": {
        "new1": {
            "pmc": 35,
            "keV": 5.5
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path), Body: bytes.NewReader([]byte(`{
    "peaks": {
        "new2": {
            "pmc": 35,
            "keV": 5.5
        }
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path), Body: bytes.NewReader([]byte(`{
    "peaks": {
        "id-1": {
            "pmc": 32,
            "keV": 5.6
        },
        "id-2": {
            "pmc": 44,
            "keV": 7.7
        },
        "new3": {
            "pmc": 35,
            "keV": 5.5
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"new1", "new2", "new3", "new4"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	postItem := `{
	"pmc": 35,
	"keV": 5.5
}`

	// File missing, first go, should just create
	req, _ := http.NewRequest("POST", "/diffraction/manual/rtt-123", bytes.NewReader([]byte(postItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Should ignore the fact that the incoming file is garbage, and write a new one
	req, _ = http.NewRequest("POST", "/diffraction/manual/rtt-123", bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// New appended to existing list
	req, _ = http.NewRequest("POST", "/diffraction/manual/rtt-123", bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "new1": {
	//         "pmc": 35,
	//         "keV": 5.5
	//     }
	// }
	//
	// 200
	// {
	//     "new2": {
	//         "pmc": 35,
	//         "keV": 5.5
	//     }
	// }
	//
	// 200
	// {
	//     "id-1": {
	//         "pmc": 32,
	//         "keV": 5.6
	//     },
	//     "id-2": {
	//         "pmc": 44,
	//         "keV": 7.7
	//     },
	//     "new3": {
	//         "pmc": 35,
	//         "keV": 5.5
	//     }
	// }
}

func Example_diffractionHandler_DeleteManual() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`garbage`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(manualFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(manualFile))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(manualS3Path), Body: bytes.NewReader([]byte(`{
    "peaks": {
        "id-1": {
            "pmc": 32,
            "keV": 5.6
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File missing, 404
	req, _ := http.NewRequest("DELETE", "/diffraction/manual/rtt-123/new-1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Incoming is garbage, 500
	req, _ = http.NewRequest("DELETE", "/diffraction/manual/rtt-123/new-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Not found in list
	req, _ = http.NewRequest("DELETE", "/diffraction/manual/rtt-123/new-3", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// OK
	req, _ = http.NewRequest("DELETE", "/diffraction/manual/rtt-123/id-2", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// new-1 not found
	//
	// 500
	// invalid character 'g' looking for beginning of value
	//
	// 404
	// new-3 not found
	//
	// 200
	// {
	//     "id-1": {
	//         "pmc": 32,
	//         "keV": 5.6
	//     }
	// }
}
