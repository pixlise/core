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

	"github.com/pixlise/core/v2/core/awsutil"

	expressionDB "github.com/pixlise/core/v2/core/expressions/database"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

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

// NOTE: the "OK" case is so simple, it'll be writing the same test out as Test_Module_DB_Create
