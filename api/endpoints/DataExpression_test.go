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
	"net/http"
	"testing"

	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/awsutil"
	expressionDB "github.com/pixlise/core/v3/core/expressions/database"
	"github.com/pixlise/core/v3/core/expressions/expressions"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/timestamper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

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
			"abc123", "Calcium weight%", "element(\"Ca\", \"%\")", "PIXLANG", "comments for abc123 expression", []string{"latest"},
			[]expressions.ModuleReference{
				{"mod123", "2.3.4"},
			},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", false, 1668100000, 1668100000),
			&expressions.DataExpressionExecStats{
				[]string{"Ca", "Fe"},
				340.5,
				1234568888,
			},
		},
		{
			"def456", "Iron Error", "element(\"Fe\", \"err\")", "PIXLANG", "comments for def456 expression", []string{}, []expressions.ModuleReference{},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", false, 1668100001, 1668100001),
			nil,
		},
		{
			"ghi789", "Iron %", "element(\"Fe\", \"%\")", "PIXLANG", "", []string{}, []expressions.ModuleReference{},
			makeOrigin("999", "Peter N", "niko@spicule.co.uk", true, 1668100002, 1668100002),
			&expressions.DataExpressionExecStats{
				[]string{"Na", "Ti"},
				20,
				1234568999,
			},
		},
		// Same as first item, but with real user id and different ID
		{
			"abc111", "Calcium weight%", "element(\"Ca\", \"%\")", "PIXLANG", "comments for abc111 expression", []string{}, []expressions.ModuleReference{},
			makeOrigin("600f2a0806b6c70071d3d174", "Peter N", "niko@spicule.co.uk", false, 1668100000, 1668100000),
			&expressions.DataExpressionExecStats{
				[]string{"Ca", "Fe"},
				340,
				1234568888,
			},
		},
		// Same as above item, but shared
		{
			"abc111", "Calcium weight%", "element(\"Ca\", \"%\")", "PIXLANG", "comments for abc111 expression", []string{}, []expressions.ModuleReference{},
			makeOrigin("600f2a0806b6c70071d3d174", "Peter N", "niko@spicule.co.uk", true, 1668100000, 1668100000),
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
        "id": "abc123",
        "name": "Calcium weight%",
        "sourceCode": "",
        "sourceLanguage": "PIXLANG",
        "comments": "comments for abc123 expression",
        "tags": [
            "latest"
        ],
        "moduleReferences": [
            {
                "moduleID": "mod123",
                "version": "2.3.4"
            }
        ],
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
            "runtimeMs": 340.5,
            "mod_unix_time_sec": 1234568888
        }
    },
    "def456": {
        "id": "def456",
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
        "id": "shared-ghi789",
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
    "id": "abc123",
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "PIXLANG",
    "comments": "comments for abc123 expression",
    "tags": [
        "latest"
    ],
    "moduleReferences": [
        {
            "moduleID": "mod123",
            "version": "2.3.4"
        }
    ],
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
        "runtimeMs": 340.5,
        "mod_unix_time_sec": 1234568888
    }
}
`)

		req, _ = http.NewRequest("GET", "/data-expression/shared-ghi789", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "id": "shared-ghi789",
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
    "id": "id16",
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
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.3.4"
		}
	]
}`

		// OK
		req, _ := http.NewRequest("PUT", "/data-expression/abc111", bytes.NewReader([]byte(putItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "id": "abc111",
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "LUA",
    "comments": "comments for abc123 expression",
    "tags": [
        "newest"
    ],
    "moduleReferences": [
        {
            "moduleID": "mod123",
            "version": "2.3.4"
        }
    ],
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

		const putItemBadModule = `{
	"name": "Calcium weight%",
	"sourceCode": "element(\"Ca\", \"%\")",
	"sourceLanguage": "LUA",
	"comments": "comments for abc123 expression",
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.three.4"
		}
	]
}`
		// Bad module version specified
		req, _ = http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItemBadModule)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `Invalid version for module: mod123. Error was: 2.three.4
`)

		const putItemDuplicateModule = `{
	"name": "Calcium weight%",
	"sourceCode": "element(\"Ca\", \"%\")",
	"sourceLanguage": "LUA",
	"comments": "comments for abc123 expression",
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.3.4"
		},
		{
			"moduleID": "mod123",
			"version": "2.4.4"
		}
	]
}`
		// Bad module version specified
		req, _ = http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItemDuplicateModule)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `Duplicate modules: mod123
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_Put_NoSourceCode(t *testing.T) {
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
			// GET item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(3, true),
			),
			// PUT success
			mtest.CreateSuccessResponse(),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100002, 1668100003},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Act as an update, if input doesn't contain source code, we look for the existing item
		// and preserve the old source field
		const putItemNoSource = `{
	"name": "Calcium weight%",
	"sourceLanguage": "LUA",
	"comments": "comments for abc123 expression",
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.3.4"
		}
	]
}`
		req, _ := http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItemNoSource)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "id": "abc123",
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "LUA",
    "comments": "comments for abc123 expression",
    "tags": [
        "newest"
    ],
    "moduleReferences": [
        {
            "moduleID": "mod123",
            "version": "2.3.4"
        }
    ],
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100002
}
`)

		const putItemBlankSource = `{
	"name": "Calcium weight%",
	"sourceCode": "",
	"sourceLanguage": "LUA",
	"comments": "comments for abc123 expression",
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.3.4"
		}
	]
}`

		req, _ = http.NewRequest("PUT", "/data-expression/abc123", bytes.NewReader([]byte(putItemBlankSource)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "id": "abc123",
    "name": "Calcium weight%",
    "sourceCode": "element(\"Ca\", \"%\")",
    "sourceLanguage": "LUA",
    "comments": "comments for abc123 expression",
    "tags": [
        "newest"
    ],
    "moduleReferences": [
        {
            "moduleID": "mod123",
            "version": "2.3.4"
        }
    ],
    "shared": false,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100003
}
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_Put_Shared(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET not found
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
			// GET found (existing, not owned by user)
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(2, true),
			),
			// GET found (existing)
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(4, true),
			),
			// PUT success
			mtest.CreateSuccessResponse(),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100004},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Act as an update, if input doesn't contain source code, we look for the existing item
		// and preserve the old source field
		const putItem = `{
	"name": "Calcium weight %",
    "sourceCode": "element(\"CaO\", \"%\")",
	"sourceLanguage": "LUA",
	"comments": "comments for abc111 expression new",
	"tags": ["newest"],
	"moduleReferences": [
		{
			"moduleID": "mod123",
			"version": "2.3.4"
		}
	]
}`

		// Editing without shared prefix on ID should fail
		req, _ := http.NewRequest("PUT", "/data-expression/abc111", bytes.NewReader([]byte(putItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 404, `abc111 not found
`)

		// Editing one not owned by caller, should fail
		req, _ = http.NewRequest("PUT", "/data-expression/ghi789", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `cannot edit expression not owned by user
`)

		req, _ = http.NewRequest("PUT", "/data-expression/shared-abc111", bytes.NewReader([]byte(putItem)))
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "id": "shared-abc111",
    "name": "Calcium weight %",
    "sourceCode": "element(\"CaO\", \"%\")",
    "sourceLanguage": "LUA",
    "comments": "comments for abc111 expression new",
    "tags": [
        "newest"
    ],
    "moduleReferences": [
        {
            "moduleID": "mod123",
            "version": "2.3.4"
        }
    ],
    "shared": true,
    "creator": {
        "name": "Niko Bellic",
        "user_id": "600f2a0806b6c70071d3d174",
        "email": "niko@spicule.co.uk"
    },
    "create_unix_time_sec": 1668100000,
    "mod_unix_time_sec": 1668100004
}
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ExecStatPut(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
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
			QueuedTimeStamps: []int64{1668100001, 1668100002},
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

		checkResult(t, resp, 200, `{
    "dataRequired": [
        "Ca",
        "Fe"
    ],
    "runtimeMs": 340,
    "mod_unix_time_sec": 1668100001
}
`)

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

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_DeleteNotFound(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET returns no existing item
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

		// Not found
		req, _ := http.NewRequest("DELETE", "/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 404, `abc999 not found
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_DeleteNoPermission(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET item
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

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Not found
		req, _ := http.NewRequest("DELETE", "/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 401, `abc999 not owned by 600f2a0806b6c70071d3d174
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_DeleteOK(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(3, false),
			),
			// DELETE success
			mtest.CreateSuccessResponse(),
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

		// Not found
		req, _ := http.NewRequest("DELETE", "/data-expression/abc111", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 200, `"abc111"
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ShareNotFound(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET returns no existing item
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

		// Not found
		req, _ := http.NewRequest("POST", "/share/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 404, `abc999 not found
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ShareNoPermissions(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET item
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

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Not found
		req, _ := http.NewRequest("POST", "/share/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 401, `abc999 not owned by 600f2a0806b6c70071d3d174
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ShareAlreadyShared(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// GET item
			mtest.CreateCursorResponse(
				0,
				"expressions-unit_test.expressions",
				mtest.FirstBatch,
				makeExprDBList(4, true),
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100000},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Not found
		req, _ := http.NewRequest("POST", "/share/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 400, `abc999 already shared
`)
	})
}

// NOTE: Major flaw here is that we can't "check" what the DB write looks like!
func Test_dataExpressionHandler_ShareOK(t *testing.T) {
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
			// Success for the share
			mtest.CreateSuccessResponse(),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		idGen := services.MockIDGenerator{
			IDs: []string{"ddd222"},
		}
		svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1668100008},
		}
		envName := "unit_test"
		svcs.Mongo = mt.Client
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, envName)
		svcs.Expressions = expressionDB.MakeExpressionDB(envName, &svcs)
		apiRouter := MakeRouter(svcs)

		// Not found
		req, _ := http.NewRequest("POST", "/share/data-expression/abc999", nil)
		resp := executeRequest(req, apiRouter.Router)
		checkResult(t, resp, 200, `"ddd222"
`)
	})
}
