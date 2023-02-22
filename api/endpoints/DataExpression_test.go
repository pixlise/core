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
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

const exprS3Path = "UserContent/600f2a0806b6c70071d3d174/DataExpressions.json"
const exprSharedS3Path = "UserContent/shared/DataExpressions.json"
const singleExprFile = `{
	"abc123": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "ContextImage",
		"comments": "comments for abc123 expression",
		"tags": [],
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		},
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
	}
}`
const exprFile = `{
	"abc123": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "ContextImage",
		"comments": "comments for abc123 expression",
		"tags": [],
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		},
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
	},
	"def456": {
		"name": "Iron Error",
		"expression": "element(\"Fe\", \"err\")",
		"type": "BinaryPlot",
		"comments": "comments for def456 expression",
		"tags": [],
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		},
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
	}
}`
const sharedExprFile = `{
	"expression-1": {
		"name": "Calcium weight%",
		"expression": "element(\"Ca\", \"%\")",
		"type": "ContextImage",
		"comments": "comments for shared-expression-1 expression",
		"tags": [],
		"creator": {
			"user_id": "999",
			"name": "Peter N",
            "email": "niko@spicule.co.uk"
		},
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
	}
}`

func Test_dataExpressionHandler_List(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "999"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Peter N"},
						{"Email", "peter@spicule.co.uk"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()
		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
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
				Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
			},
			{
				// Note: No comments!
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
		"ghi789": {
			"name": "Iron %",
			"expression": "element(\"Fe\", \"%\")",
			"type": "TernaryPlot",
			"creator": {
				"user_id": "999",
				"name": "Peter N",
				"email": "niko@spicule.co.uk"
			}
		}
	}`))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/data-expression", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/data-expression", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "abc123": {
        "name": "Calcium weight%",
        "expression": "",
        "type": "ContextImage",
        "comments": "comments for abc123 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "def456": {
        "name": "Iron Error",
        "expression": "",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "shared-ghi789": {
        "name": "Iron %",
        "expression": "",
        "type": "TernaryPlot",
        "comments": "",
        "tags": [],
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        }
    }
}
`)
	})
}

func Test_dataExpressionHandler_Get(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "999"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Peter N"},
						{"Email", "peter@spicule.co.uk"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()
		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
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
				Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
			},
			{
				// Note: No comments!
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
		"ghi789": {
			"name": "Iron %",
			"expression": "element(\"Fe\", \"%\")",
			"type": "TernaryPlot",
			"creator": {
				"user_id": "999",
				"name": "Peter N",
				"email": "niko@spicule.co.uk"
			}
		}
	}`))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression/abc123", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `abc123 not found
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `shared-ghi789 not found
`)

		req, _ = http.NewRequest("GET", "/data-expression/abc123", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `abc123 not found
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `shared-ghi789 not found
`)

		req, _ = http.NewRequest("GET", "/data-expression/abc123", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Calcium weight%",
    "expression": "element(\"Ca\", \"%\")",
    "type": "ContextImage",
    "comments": "comments for abc123 expression",
    "tags": [],
    "shared": false,
    "creator": {
        "name": "Peter N",
        "user_id": "999",
        "email": "peter@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100000
}
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Iron %",
    "expression": "element(\"Fe\", \"%\")",
    "type": "TernaryPlot",
    "comments": "",
    "tags": [],
    "shared": true,
    "creator": {
        "name": "Peter N",
        "user_id": "999",
        "email": "peter@spicule.co.uk"
    }
}
`)
	})
}

func Example_dataExpressionHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "id16": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142579,
        "mod_unix_time_sec": 1668142579
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "id17": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142580,
        "mod_unix_time_sec": 1668142580
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "id18": {
        "name": "Sodium weight%",
        "expression": "element(\"Na\", \"%\")",
        "type": "ContextImage",
        "comments": "sodium comment here",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668142581,
        "mod_unix_time_sec": 1668142581
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
	idGen.ids = []string{"id16", "id17", "id18"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579, 1668142580, 1668142581},
	}
	apiRouter := MakeRouter(svcs)

	const putItem = `{
	"name": "Sodium weight%",
	"expression": "element(\"Na\", \"%\")",
	"type": "ContextImage",
	"comments": "sodium comment here",
	"tags": []
}`

	// File not in S3, should work
	req, _ := http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already contains stuff, this is added
	req, _ = http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "id16": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
	//         "tags": [],
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142579,
	//         "mod_unix_time_sec": 1668142579
	//     }
	// }
	//
	// 200
	// {
	//     "id17": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
	//         "tags": [],
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142580,
	//         "mod_unix_time_sec": 1668142580
	//     }
	// }
	//
	// 200
	// {
	//     "id18": {
	//         "name": "Sodium weight%",
	//         "expression": "element(\"Na\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "sodium comment here",
	//         "tags": [],
	//         "shared": false,
	//         "creator": {
	//             "name": "Niko Bellic",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668142581,
	//         "mod_unix_time_sec": 1668142581
	//     }
	// }
}

func Example_dataExpressionHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(singleExprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(singleExprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExprFile))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
        "name": "Calcium weight%",
        "expression": "element(\"Ca\", \"%\")",
        "type": "ContextImage",
        "comments": "comments for abc123 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668100000},
	}
	apiRouter := MakeRouter(svcs)

	const putItem = `{
		"name": "Calcium weight%",
        "expression": "element(\"Ca\", \"%\")",
        "type": "ContextImage",
        "comments": "comments for abc123 expression",
        "tags": []
	}`

	// File not in S3, not found
	req, _ := http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, not found
	req, _ = http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already this id, should overwrite
	req, _ = http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain this id, not found
	req, _ = http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Can't edit shared ids
	req, _ = http.NewRequest("PUT", "/data-expression/shared-expression-1", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// aaa111 not found
	//
	// 404
	// aaa111 not found
	//
	// 200
	// {
	//     "abc123": {
	//         "name": "Calcium weight%",
	//         "expression": "element(\"Ca\", \"%\")",
	//         "type": "ContextImage",
	//         "comments": "comments for abc123 expression",
	//         "tags": [],
	//         "shared": false,
	//         "creator": {
	//             "name": "Peter N",
	//             "user_id": "999",
	//             "email": "niko@spicule.co.uk"
	//         },
	//         "create_unix_time_sec": 1668100000,
	//         "mod_unix_time_sec": 1668100000
	//     }
	// }
	//
	// 404
	// aaa111 not found
	//
	// 400
	// cannot edit shared expression not owned by user
}

func Example_dataExpressionHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "def456": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path), Body: bytes.NewReader([]byte(`{
    "def456": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    }
}`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/abc999", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/data-expression/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/data-expression/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/data-expression/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// abc123 not found
	//
	// 404
	// abc123 not found
	//
	// 404
	// abc999 not found
	//
	// 200
	// {
	//     "abc123": "abc123"
	// }
	//
	// 401
	// def456 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
	// {
	//     "shared-def456": "shared-def456"
	// }
}

func Example_dataExpressionHandler_Share() {
	sharedExpressionsContents := `{
		"aaa333": {
			"name": "Calcium Error",
			"expression": "element(\"Ca\", \"err\")",
			"type": "TernaryPlot",
			"comments": "calcium comments",
			"tags": [],
			"shared": false,
			"creator": {
				"name": "The sharer",
				"user_id": "600f2a0806b6c70071d3d174",
				"email": "niko@spicule.co.uk"
			},
			"create_unix_time_sec": 1668150001,
			"mod_unix_time_sec": 1668150001
		}
	}`
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(exprFile))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(sharedExpressionsContents))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(exprSharedS3Path), Body: bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "Calcium Error",
        "expression": "element(\"Ca\", \"err\")",
        "type": "TernaryPlot",
        "comments": "calcium comments",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668150001,
        "mod_unix_time_sec": 1668150001
    },
    "ddd222": {
        "name": "Iron Error",
        "expression": "element(\"Fe\", \"err\")",
        "type": "BinaryPlot",
        "comments": "comments for def456 expression",
        "tags": [],
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668142579
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"ddd222"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User file not there, should say not found
	req, _ := http.NewRequest("POST", "/share/data-expression/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/data-expression/abc123", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/data-expression/zzz222", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/data-expression/def456", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// abc123 not found
	//
	// 404
	// abc123 not found
	//
	// 404
	// zzz222 not found
	//
	// 200
	// "ddd222"
}
