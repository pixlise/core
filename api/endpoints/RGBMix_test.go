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

const rgbMixUserS3Path = "UserContent/600f2a0806b6c70071d3d174/RGBMixes.json"
const rgbMixSharedS3Path = "UserContent/shared/RGBMixes.json"
const rgbMixFileData = `{
	"abc123": {
		"name": "Ca-Ti-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
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
		"name": "Ca-Fe-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.4,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.4,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 3.4,
			"rangeMax": 6.3
		},
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
const rgbMixSharedFileData = `{
	"111": {
		"name": "Ca-Ti-Al ratios",
		"red": {
			"expressionID": "expr-for-Ca",
			"rangeMin": 1.5,
			"rangeMax": 4.3
		},
		"green": {
			"expressionID": "expr-for-Al",
			"rangeMin": 2.5,
			"rangeMax": 5.3
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3.5,
			"rangeMax": 6.3
		},
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

func Test_RGBMixHandler_List(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// 999 - not found
			mtest.CreateCursorResponse(
				1,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.NextBatch,
			),
			// 999 - not found (again)
			mtest.CreateCursorResponse(
				1,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.NextBatch,
			),
			// User 88
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "88"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Agent 88"},
						{"Email", "agent_88@spicule.co.uk"},
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
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
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
				Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
			},
			// Shared items, NOTE this returns an old-style "element" for checking backwards compatibility!
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
		"ghi789": {
			"name": "Na-Fe-Al ratios",
			"red": {
				"expressionID": "expr-for-Na",
				"rangeMin": 1,
				"rangeMax": 2
			},
			"green": {
				"expressionID": "expr-for-Al",
				"rangeMin": 2,
				"rangeMax": 5
			},
			"blue": {
				"element": "Fe",
				"rangeMin": 3,
				"rangeMax": 6
			},
			"creator": {
				"user_id": "88",
				"name": "88",
				"email": "mr88@spicule.co.uk"
			},
			"create_unix_time_sec": 1668100002,
			"mod_unix_time_sec": 1668100002
		}
	}`))),
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/rgb-mix", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/rgb-mix", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)

		req, _ = http.NewRequest("GET", "/rgb-mix", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "abc123": {
        "name": "Ca-Ti-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.5,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.5,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3.5,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "shared-ghi789": {
        "name": "Na-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-elem-Fe-%",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "tags": [],
        "shared": true,
        "creator": {
            "name": "Agent 88",
            "user_id": "88",
            "email": "agent_88@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100002,
        "mod_unix_time_sec": 1668100002
    }
}
`)
	})
}

func Example_RGBMixHandler_Get() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// POST not implemented! Should return 405
	req, _ := http.NewRequest("GET", "/rgb-mix/abc123", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
}

func Example_RGBMixHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "rgbmix-id16": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "rgbmix-id17": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
        "name": "Ca-Ti-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.5,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.5,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3.5,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    },
    "rgbmix-id18": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
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
	"name": "Sodium and stuff",
	"red": {
		"expressionID": "expr-for-Na",
		"rangeMin": 1,
		"rangeMax": 2
	},
	"green": {
		"expressionID": "expr-for-Fe",
		"rangeMin": 2,
		"rangeMax": 5
	},
	"blue": {
		"expressionID": "expr-for-Ti",
		"rangeMin": 3,
		"rangeMax": 6
	},
	"tags": []
}`
	const putItemWithElement = `{
	"name": "Sodium and stuff",
	"red": {
		"expressionID": "expr-for-Na",
		"rangeMin": 1,
		"rangeMax": 2
	},
	"green": {
		"element": "Fe",
		"rangeMin": 2,
		"rangeMax": 5
	},
	"blue": {
		"expressionID": "expr-for-Ti",
		"rangeMin": 3,
		"rangeMax": 6
	},
	"tags": []
}`

	// File not in S3, should work
	req, _ := http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should work
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already contains stuff, this is added
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Adding old-style with element defined, should fail
	req, _ = http.NewRequest("POST", "/rgb-mix", bytes.NewReader([]byte(putItemWithElement)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
	// 200
	//
	// 200
	//
	// 400
	// RGB Mix definition with elements is deprecated
}

func Example_RGBMixHandler_Put() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixSharedFileData))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "abc123": {
        "name": "Ca-Ti-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.5,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.5,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3.5,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000
    },
    "def456": {
        "name": "Sodium and stuff",
        "red": {
            "expressionID": "expr-for-Na",
            "rangeMin": 1,
            "rangeMax": 2
        },
        "green": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 2,
            "rangeMax": 5
        },
        "blue": {
            "expressionID": "expr-for-Ti",
            "rangeMin": 3,
            "rangeMax": 6
        },
        "tags": [],
        "shared": false,
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1668142579},
	}
	apiRouter := MakeRouter(svcs)

	const putItem = `{
		"name": "Sodium and stuff",
		"red": {
			"expressionID": "expr-for-Na",
			"rangeMin": 1,
			"rangeMax": 2
		},
		"green": {
			"expressionID": "expr-for-Fe",
			"rangeMin": 2,
			"rangeMax": 5
		},
		"blue": {
			"expressionID": "expr-for-Ti",
			"rangeMin": 3,
			"rangeMax": 6
		},
		"tags": []
	}`

	// File not in S3, not found
	req, _ := http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, not found
	req, _ = http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File already this id, should overwrite
	req, _ = http.NewRequest("PUT", "/rgb-mix/def456", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File doesn't contain this id, not found
	req, _ = http.NewRequest("PUT", "/rgb-mix/aaa111", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Can't edit shared ids
	req, _ = http.NewRequest("PUT", "/rgb-mix/shared-111", bytes.NewReader([]byte(putItem)))
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
	//
	// 404
	// aaa111 not found
	//
	// 400
	// cannot edit shared RGB mixes created by others
}

func Example_RGBMixHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": false,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100001,
        "mod_unix_time_sec": 1668100001
    }
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path), Body: bytes.NewReader([]byte(`{
    "def456": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
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
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path), Body: bytes.NewReader([]byte(`{}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Delete finds file missing, ERROR
	req, _ := http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds empty file, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete cant find item, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc999", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete finds item, OK
	req, _ = http.NewRequest("DELETE", "/rgb-mix/abc123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item but from wrong user, ERROR
	req, _ = http.NewRequest("DELETE", "/rgb-mix/shared-def456", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Delete shared item, OK
	req, _ = http.NewRequest("DELETE", "/rgb-mix/shared-def456", nil)
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
	//
	// 401
	// def456 not owned by 600f2a0806b6c70071d3d174
	//
	// 200
}

func Example_RGBMixHandler_Share() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
		// Reading shared file to add to it
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(rgbMixFileData))),
		},
		// Shared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": true,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100006,
        "mod_unix_time_sec": 1668100006
    }
}`))),
		},
	}
	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixSharedS3Path), Body: bytes.NewReader([]byte(`{
    "aaa333": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "tags": [],
        "shared": true,
        "creator": {
            "name": "The sharer",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100006,
        "mod_unix_time_sec": 1668100006
    },
    "rgbmix-ddd222": {
        "name": "Ca-Fe-Al ratios",
        "red": {
            "expressionID": "expr-for-Ca",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "expr-for-Al",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "expr-for-Fe",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
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
	req, _ := http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File empty in S3, should say not found
	req, _ = http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File missing the id being shared
	req, _ = http.NewRequest("POST", "/share/rgb-mix/zzz222", bytes.NewReader([]byte(putItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains ID, share OK
	req, _ = http.NewRequest("POST", "/share/rgb-mix/def456", bytes.NewReader([]byte(putItem)))
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
	// "rgbmix-ddd222"
}

func Example_RGBMixHandler_Share_UnsharedExprs() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(rgbMixUserS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "abc123": {
        "name": "K-Al-Fe already shared",
        "red": {
            "expressionID": "expr-for-K",
            "rangeMin": 1.4,
            "rangeMax": 4.3
        },
        "green": {
            "expressionID": "shared-abcd123",
            "rangeMin": 2.4,
            "rangeMax": 5.3
        },
        "blue": {
            "expressionID": "xyz123",
            "rangeMin": 3.4,
            "rangeMax": 6.3
        },
        "shared": false,
        "creator": {
            "name": "Niko",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        }
    }
}`))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	const putItem = ""

	// User trying to share RGB mix with non-shared expressions, should fail
	req, _ := http.NewRequest("POST", "/share/rgb-mix/abc123", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// When sharing RGB mix, it must only reference shared expressions
}
