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
	"github.com/pixlise/core/v2/core/awsutil"
)

const elemUserS3Path = "UserContent/600f2a0806b6c70071d3d174/ElementSets.json"
const elemSharedS3Path = "UserContent/shared/ElementSets.json"
const elemFile = `{
	"13": {
		"name": "My Monday Elements",
		"lines": [
			{
				"Z": 26,
				"K": true,
				"L": true,
				"M": false,
				"Esc": false
			},
			{
				"Z": 20,
				"K": true,
				"L": false,
				"M": false,
				"Esc": false
			}
		],
		"creator": { "name": "Peter", "user_id": "u123" }
	},
	"44": {
		"name": "My Tuesday Elements",
		"lines": [
			{
				"Z": 13,
				"K": true,
				"L": false,
				"M": false,
				"Esc": false
			},
			{
				"Z": 14,
				"K": true,
				"L": false,
				"M": false,
				"Esc": false
			}
		],
		"creator": { "name": "Tom", "user_id": "u124" }
	}
}`

func Example_elementSetHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil, // No file in S3
		nil, // No file in S3
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"13": {
		"name": "My Monday Elements",
		"lines": [
			{
				"Z": 26,
				"K": true,
				"L": true,
				"M": false,
				"Esc": false
			},
			{
				"Z": 20,
				"K": true,
				"L": false,
				"M": false,
				"Esc": false
			}
		],
		"creator": { "name": "Peter", "user_id": "u123" }
	}
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"88": {
		"name": "Shared Elements",
		"lines": [
			{
				"Z": 32,
				"K": true,
				"L": true,
				"M": false,
				"Esc": false
			}
		],
		"creator": { "name": "Mike", "user_id": "u125" }
	}
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/element-set", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/element-set", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("GET", "/element-set", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {}
	//
	// 200
	// {}
	//
	// 200
	// {
	//     "13": {
	//         "name": "My Monday Elements",
	//         "atomicNumbers": [
	//             26,
	//             20
	//         ],
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter",
	//             "user_id": "u123",
	//             "email": ""
	//         }
	//     },
	//     "shared-88": {
	//         "name": "Shared Elements",
	//         "atomicNumbers": [
	//             32
	//         ],
	//         "shared": true,
	//         "creator": {
	//             "name": "Mike",
	//             "user_id": "u125",
	//             "email": ""
	//         }
	//     }
	// }
}

func Example_elementSetHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File not in S3, should return 404
	req, _ := http.NewRequest("GET", "/element-set/13", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File in S3 empty, should return 404
	req, _ = http.NewRequest("GET", "/element-set/13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains stuff, using ID thats not in there, should return 404
	req, _ = http.NewRequest("GET", "/element-set/15", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains stuff, using ID that exists
	req, _ = http.NewRequest("GET", "/element-set/13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Check that shared file was loaded if shared ID sent in
	req, _ = http.NewRequest("GET", "/element-set/shared-13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 13 not found
	//
	// 404
	// 13 not found
	//
	// 404
	// 15 not found
	//
	// 200
	// {
	//     "name": "My Monday Elements",
	//     "lines": [
	//         {
	//             "Z": 26,
	//             "K": true,
	//             "L": true,
	//             "M": false,
	//             "Esc": false
	//         },
	//         {
	//             "Z": 20,
	//             "K": true,
	//             "L": false,
	//             "M": false,
	//             "Esc": false
	//         }
	//     ],
	//     "shared": false,
	//     "creator": {
	//         "name": "Peter",
	//         "user_id": "u123",
	//         "email": ""
	//     }
	// }
	//
	// 200
	// {
	//     "name": "My Monday Elements",
	//     "lines": [
	//         {
	//             "Z": 26,
	//             "K": true,
	//             "L": true,
	//             "M": false,
	//             "Esc": false
	//         },
	//         {
	//             "Z": 20,
	//             "K": true,
	//             "L": false,
	//             "M": false,
	//             "Esc": false
	//         }
	//     ],
	//     "shared": true,
	//     "creator": {
	//         "name": "Peter",
	//         "user_id": "u123",
	//         "email": ""
	//     }
	// }
}

func Example_elementSetHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path), Body: bytes.NewReader([]byte(`{
    "55": {
        "name": "Latest set",
        "lines": [
            {
                "Z": 43,
                "K": true,
                "L": true,
                "M": true,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path), Body: bytes.NewReader([]byte(`{
    "56": {
        "name": "Latest set",
        "lines": [
            {
                "Z": 43,
                "K": true,
                "L": true,
                "M": true,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path), Body: bytes.NewReader([]byte(`{
    "13": {
        "name": "My Monday Elements",
        "lines": [
            {
                "Z": 26,
                "K": true,
                "L": true,
                "M": false,
                "Esc": false
            },
            {
                "Z": 20,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": ""
        }
    },
    "44": {
        "name": "My Tuesday Elements",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            },
            {
                "Z": 14,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        }
    },
    "57": {
        "name": "Latest set",
        "lines": [
            {
                "Z": 43,
                "K": true,
                "L": true,
                "M": true,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"55", "56", "57"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const postItem = `{
	"name": "Latest set",
	"lines": [
		{
			"Z": 43,
			"K": true,
			"L": true,
			"M": true,
			"Esc": false
		}
	]
}`

	// File not in S3, should work
	req, _ := http.NewRequest("POST", "/element-set", bytes.NewReader([]byte(postItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", "/element-set", bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain item by this name, should work (add)
	req, _ = http.NewRequest("POST", "/element-set", bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
	// 200
	//
	// 200
}

func Example_elementSetHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path), Body: bytes.NewReader([]byte(`{
    "13": {
        "name": "My Monday Elements",
        "lines": [
            {
                "Z": 26,
                "K": true,
                "L": true,
                "M": false,
                "Esc": false
            },
            {
                "Z": 20,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Peter",
            "user_id": "u123",
            "email": ""
        }
    },
    "44": {
        "name": "Latest set",
        "lines": [
            {
                "Z": 43,
                "K": true,
                "L": true,
                "M": true,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = `{
		"name": "Latest set",
		"lines": [
			{
				"Z": 43,
				"K": true,
				"L": true,
				"M": true,
				"Esc": false
			}
		]
	}`

	// File not in S3, should say not found
	req, _ := http.NewRequest("PUT", "/element-set/44", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("PUT", "/element-set/44", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already contains this id, should overwrite
	req, _ = http.NewRequest("PUT", "/element-set/44", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain this id, should say not found
	req, _ = http.NewRequest("PUT", "/element-set/59", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Can't edit shared ids
	req, _ = http.NewRequest("PUT", "/element-set/shared-59", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 44 not found
	//
	// 404
	// 44 not found
	//
	// 200
	//
	// 404
	// 59 not found
	//
	// 400
	// Cannot edit shared items
}

func Example_elementSetHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "55": {
        "name": "The shared item",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": ""
        }
    }
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path), Body: bytes.NewReader([]byte(`{
    "44": {
        "name": "My Tuesday Elements",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            },
            {
                "Z": 14,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        }
    }
}`)),
		},
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", "/element-set/13", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/element-set/13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/element-set/15", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/element-set/13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/element-set/shared-13", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/element-set/shared-55", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 13 not found
	//
	// 404
	// 13 not found
	//
	// 404
	// 15 not found
	//
	// 200
	//
	// 401
	// 13 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
}

func Example_elementSetHandler_Share() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(elemUserS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/ElementSets.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(elemFile))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "55": {
        "name": "Already shared item",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174"
        }
    }
}`))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		s3.PutObjectInput{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/ElementSets.json"), Body: bytes.NewReader([]byte(`{
    "55": {
        "name": "Already shared item",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": false,
        "creator": {
            "name": "The user who shared",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": ""
        }
    },
    "77": {
        "name": "My Tuesday Elements",
        "lines": [
            {
                "Z": 13,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            },
            {
                "Z": 14,
                "K": true,
                "L": false,
                "M": false,
                "Esc": false
            }
        ],
        "shared": true,
        "creator": {
            "name": "Tom",
            "user_id": "u124",
            "email": ""
        }
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"77"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/element-set/33", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/element-set/33", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/element-set/33", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/element-set/44", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 33 not found
	//
	// 404
	// 33 not found
	//
	// 404
	// 33 not found
	//
	// 200
	// "77"
}
