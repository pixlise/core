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
	"net/http"
	"testing"

	"github.com/pixlise/core/v2/core/awsutil"

	expressionDB "github.com/pixlise/core/v2/core/expressions/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_Module_Listing(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// Modules
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.modules",
				mtest.FirstBatch,
				bson.D{
					{"_id", "mod123"},
					{"name", "Module1"},
					{"comments", "Module 1"},
					{"origin", bson.D{
						{"shared", true},
						{"creator", bson.D{
							{"name", "Peter N"},
							{"userid", "999"},
							{"email", "peter@pixlise.org"},
						}},
						{"CreatedUnixTimeSec", 1234567890},
						{"ModifiedUnixTimeSec", 1234567891},
					}},
				}),
			// Module 1 versions
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.moduleVersions",
				mtest.FirstBatch,
				bson.D{
					{"moduleID", "mod123"},
					{"sourceCode", "element(\"Ca\", \"%\", \"A\")"}, // TODO: this shouldn't be here!
					{"comments", "Module 1"},
					{"version", bson.D{
						{"major", 0},
						{"minor", 0},
						{"patch", 1},
					}},
					{"tags", []string{"oldest", "A"}},
					{"TimeStampUnixSec", 1234567891},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(nil, nil, nil, nil)
		svcs.Mongo = mt.Client
		db := expressionDB.MakeExpressionDB("local", &svcs)

		svcs.Expressions = db
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-module", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "mod123": {
        "id": "mod123",
        "name": "Module1",
        "comments": "Module 1",
        "origin": {
            "shared": true,
            "creator": {
                "name": "Peter N",
                "user_id": "999",
                "email": "peter@pixlise.org"
            },
            "create_unix_time_sec": 1234567890,
            "mod_unix_time_sec": 1234567891
        },
        "versions": [
            {
                "version": "0.0.1",
                "tags": [
                    "oldest",
                    "A"
                ],
                "comments": "Module 1",
                "mod_unix_time_sec": 1234567891
            }
        ]
    }
}
`)
	})
}

func Test_Module_Create_NoSrc(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(nil, nil, nil, nil)
		svcs.Mongo = mt.Client
		db := expressionDB.MakeExpressionDB("local", &svcs)
		svcs.Expressions = db
		apiRouter := MakeRouter(svcs)

		postItem := `{
			"name": "MyModule",
			"sourceCode": "",
			"comments": "My first module",
			"tags": ["A tag"]
		}`

		req, _ := http.NewRequest("POST", "/data-module", bytes.NewReader([]byte(postItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `Source code field cannot be empty
`)
	})
}

func Test_Module_Create_BadModule(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(nil, nil, nil, nil)
		svcs.Mongo = mt.Client
		db := expressionDB.MakeExpressionDB("local", &svcs)
		svcs.Expressions = db
		apiRouter := MakeRouter(svcs)

		postItem := `{
			"name":       "My Module",
			"sourceCode": "1+1",
			"comments": "My first module",
			"tags": ["A tag"]
		}`

		req, _ := http.NewRequest("POST", "/data-module", bytes.NewReader([]byte(postItem)))
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `Invalid module name: My Module
`)
	})
}

// NOTE: the listing "OK" case is so simple, it'll be writing the same test out as Test_Module_DB_Create

func Test_Module_Get_BadVersion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(nil, nil, nil, nil)
		svcs.Mongo = mt.Client
		db := expressionDB.MakeExpressionDB("local", &svcs)
		svcs.Expressions = db
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/data-module/modid/1.oops.3", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 400, `Invalid version specified: Failed to parse version 1.oops.3, part oops is not a number
`)
	})
}

func Example_Module_DeleteModuleNotImplemented() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("DELETE", "/data-module/modid", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
	//
}

func Example_Module_DeleteModuleVersionNotImplemented() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("DELETE", "/data-module/modid/1.2.3", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 405
	//
}

// NOTE: the "OK" case is already tested in module tests
func Example_Module_AddVersion_BadStruct() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	putItem := `{
		"name":       "My Module",
		"sourceCode": "1+tag"]
	}`

	req, _ := http.NewRequest("PUT", "/data-module/mod1", bytes.NewReader([]byte(putItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// invalid character ']' after object key:value pair
}
