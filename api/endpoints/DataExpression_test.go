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
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
	expressionDB "github.com/pixlise/core/v2/core/expressions/database"
	"github.com/pixlise/core/v2/core/expressions/expressions"
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

func makeOrigin(id string, name string, email string, shared bool, crSec int64, modSec int64) pixlUser.APIObjectItem {
	bsonD := //orig, _ := bson.Marshal(
		pixlUser.APIObjectItem{
			Shared: shared,
			Creator: pixlUser.UserInfo{
				Name:   name,
				UserID: id,
				Email:  email,
			},
			CreatedUnixTimeSec:  crSec,
			ModifiedUnixTimeSec: modSec,
		} //,
	//)
	//bsonD := bson.D{}
	//bson.Unmarshal(orig, &bsonD)
	return bsonD
}

func makeExprDBList(idx int, includeSource bool) bson.D {
	items := []expressions.DataExpression{
		{
			"abc123", "Calcium weight%", "element(\"Ca\", \"%\")", "PIXLANG", "comments for abc123 expression", []string{},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", false, 1668100000, 1668100000),
			&expressions.DataExpressionExecStats{
				[]string{"Ca", "Fe"},
				340,
				1234568888,
			},
		},
		{
			"def456", "Iron Error", "element(\"Fe\", \"err\")", "PIXLANG", "comments for def456 expression", []string{},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", false, 1668100001, 1668100001),
			nil,
		},
		{
			"ghi789", "Iron %", "element(\"Fe\", \"%\")", "PIXLANG", "", []string{},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", true, 1668100002, 1668100002),
			&expressions.DataExpressionExecStats{
				[]string{"Na", "Ti"},
				20,
				1234568999,
			},
		},
		// Same as first item, but with real user id and different ID
		{
			"abc111", "Calcium weight%", "element(\"Ca\", \"%\")", "PIXLANG", "comments for abc111 expression", []string{},
			makeOrigin("600f2a0806b6c70071d3d174", "Peter N", "niko@spicule.co.uk", false, 1668100000, 1668100000),
			&expressions.DataExpressionExecStats{
				[]string{"Ca", "Fe"},
				340,
				1234568888,
			},
		},
	}

	item := items[idx]
	if !includeSource {
		item.SourceCode = ""
	}
	data, err := bson.Marshal(item)
	if err != nil {
		panic(err)
	}
	bsonD := bson.D{}
	err = bson.Unmarshal(data, &bsonD)
	if err != nil {
		panic(err)
	}
	return bsonD
}

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

func Test_dataExpressionHandler_List_Empty(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// No expressions
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{}
`)
	})
}

func Test_dataExpressionHandler_List_OK(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// Valid listing
			mtest.CreateCursorResponse(
				2,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(2, false),
			),
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
				makeExprDBList(1, false),
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
				makeExprDBList(0, false),
			),
			// User details
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

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "abc123": {
        "name": "Calcium weight%",
        "sourceCode": "",
        "sourceLanguage": "PIXLANG",
        "comments": "comments for abc123 expression",
        "tags": [],
        "shared": false,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100000,
        "mod_unix_time_sec": 1668100000,
        "recentExecStats": {
            "dataRequired": [
                "Ca",
                "Fe"
            ],
            "runtimeMs": 340,
            "mod_unix_time_sec": 1234568888
        }
    },
    "def456": {
        "name": "Iron Error",
        "sourceCode": "",
        "sourceLanguage": "PIXLANG",
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
        "sourceCode": "",
        "sourceLanguage": "PIXLANG",
        "comments": "",
        "tags": [],
        "shared": true,
        "creator": {
            "name": "Peter N",
            "user_id": "999",
            "email": "peter@spicule.co.uk"
        },
        "create_unix_time_sec": 1668100002,
        "mod_unix_time_sec": 1668100002,
        "recentExecStats": {
            "dataRequired": [
                "Na",
                "Ti"
            ],
            "runtimeMs": 20,
            "mod_unix_time_sec": 1234568999
        }
    }
}
`)
	})
}

func Test_dataExpressionHandler_Get_Missing(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// Not found (user)
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
			),
			// Not found (shared)
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression/abc123", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `abc123 not found
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `shared-ghi789 not found
`)
	})
}

func Test_dataExpressionHandler_Get_OK(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// User item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(0, true),
			),
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
			// Shared item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(2, true),
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-expression/abc123", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "PIXLANG",
    "comments": "comments for abc123 expression",
    "tags": [],
    "shared": false,
    "creator": {
        "name": "Peter N",
        "user_id": "999",
        "email": "peter@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100000,
    "recentExecStats": {
        "dataRequired": [
            "Ca",
            "Fe"
        ],
        "runtimeMs": 340,
        "mod_unix_time_sec": 1234568888
    }
}
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Iron %",
    "sourceCode": "element(\"Fe\", \"%\")",
    "sourceLanguage": "PIXLANG",
    "comments": "",
    "tags": [],
    "shared": true,
    "creator": {
        "name": "Peter N",
        "user_id": "999",
        "email": "peter@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100002,
    "mod_unix_time_sec": 1668100002,
    "recentExecStats": {
        "dataRequired": [
            "Na",
            "Ti"
        ],
        "runtimeMs": 20,
        "mod_unix_time_sec": 1234568999
    }
}
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_Post(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateSuccessResponse(),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		idGen := services.MockIDGenerator{
			IDs: []string{"id16"},
		}
		svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668142579},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)
		/*
			When we were storing in S3, we could check in our S3 mock that the S3 file write looked like...

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
		*/
		const putItem = `{
		"name": "Sodium weight%",
		"sourceCode": "element(\"Na\", \"%\")",
		"sourceLanguage": "LUA",
		"comments": "sodium comment here",
		"tags": []
	}`

		req, _ := http.NewRequest("POST", "/data-expression", bytes.NewReader([]byte(putItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Sodium weight%",
    "sourceCode": "element(\"Na\", \"%\")",
    "sourceLanguage": "LUA",
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
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_Put(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(3, true),
			),
			// PUT success
			mtest.CreateSuccessResponse(),
			// GET 2 returns no existing item
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
			),
			// GET 3 returns something that can't be edited by this user
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(0, true),
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		/* When we were writing to S3, we would verify that the S3 write looks something like:
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
		}*/

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100000},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		const putItem = `{
		"name": "Calcium weight%",
        "sourceCode": "element(\"Ca\", \"%\")",
        "sourceLanguage": "LUA",
        "comments": "comments for abc123 expression",
        "tags": []
	}`

		// OK
		req, _ := http.NewRequest("PUT", "/data-expression/abc111", bytes.NewReader([]byte(putItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "LUA",
    "comments": "comments for abc123 expression",
    "tags": [],
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100000
}
`)

		// DB doesn't contain this id, not found
		req, _ = http.NewRequest("PUT", "/data-expression/aaa111", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `aaa111 not found
`)

		// Can't edit shared ids
		req, _ = http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `cannot edit expression not owned by user
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ExecStatPut(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// GET item
			/*mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(3, true),
			),*/
			// PUT success
			mtest.CreateSuccessResponse(bson.E{Key: "nModified", Value: 1}, bson.E{Key: "n", Value: 1}),
			// GET 2 returns no existing item
			mtest.CreateCursorResponse(
				1,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100000, 1668100009},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Bad body, should reject quickly
		putItem := `{
			"dataRequired": ["Ca",
			"runtimeMs": 340
		}`
		req, _ := http.NewRequest("PUT", "/data-expression/execution-stat/abc111", bytes.NewReader([]byte(putItem)))
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 400, `invalid character ':' after array element
`)

		// Note: timestamp should be ignored!
		putItem = `{
			"dataRequired": ["Ca", "Fe"],
			"runtimeMs": 340,
			"mod_unix_time_sec": 1234567890
		}`

		req, _ = http.NewRequest("PUT", "/data-expression/execution-stat/aaa111", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, "")

		// Non-existant expression ID
		putItem = `{
			"dataRequired": ["Ca", "Fe"],
			"runtimeMs": 340,
			"mod_unix_time_sec": 1234567890
		}`
		req, _ = http.NewRequest("PUT", "/data-expression/execution-stat/aaa123", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `aaa123 not found
`)
	})
}

// Output:
// 400
// invalid character ':' after array element
//
// 200
//
// 200

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

	idGen := services.MockIDGenerator{
		IDs: []string{"ddd222"},
	}
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
