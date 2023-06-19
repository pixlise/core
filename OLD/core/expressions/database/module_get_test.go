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

package expressionDB

import (
	"reflect"
	"testing"

	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/expressions/modules"
	"github.com/pixlise/core/v3/core/pixlUser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_Module_DB_Get_DoesntExist(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				1,
				"modules-unit_test.modules",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.modules",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := makeMockSvcs(nil)
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		_, err := db.GetModule("mod123", &modules.SemanticVersion{Major: 1, Minor: 0, Patch: 1}, false)

		if err == nil {
			t.Error("Expected error")
		}

		if err.Error() != "mongo: no documents in result" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func Test_Module_DB_Get_MissingVersion(t *testing.T) {
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
				},
			),
			// Version is missing
			mtest.CreateCursorResponse(
				1,
				"modules-unit_test.moduleVersions",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.moduleVersions",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := makeMockSvcs(nil)
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		_, err := db.GetModule("mod123", &modules.SemanticVersion{Major: 1, Minor: 0, Patch: 0}, false)

		if err == nil {
			t.Error("Expected error")
		}

		if err.Error() != "Failed to get version: 1.0.0 for module: mod123. Error: mongo: no documents in result" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func Test_Module_DB_Get_OK(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// Module
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
				},
			),
			// Version:
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.moduleVersions",
				mtest.FirstBatch,
				bson.D{
					{"moduleID", "mod123"},
					{"sourceCode", "element(\"Ca\", \"%\", \"A\")"}, // TODO: this shouldn't be here!
					{"comments", "Module 1"},
					{"version", bson.D{
						{"major", 2},
						{"minor", 1},
						{"patch", 43},
					}},
					{"tags", []string{"oldest", "A"}},
					{"TimeStampUnixSec", 1234567891},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := makeMockSvcs(nil)
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		result, err := db.GetModule("mod123", &modules.SemanticVersion{Major: 2, Minor: 1, Patch: 43}, true)

		if err != nil {
			t.Error(err)
		}

		// Check entire returned result
		expected := modules.DataModuleSpecificVersionWire{
			DataModule: &modules.DataModule{ID: "mod123",
				Name:     "Module1",
				Comments: "Module 1",
				Origin: pixlUser.APIObjectItem{
					Shared:              true,
					Creator:             pixlUser.UserInfo{Name: "Peter N", UserID: "999", Email: "peter@pixlise.org"},
					CreatedUnixTimeSec:  1234567890,
					ModifiedUnixTimeSec: 1234567891,
				},
			},
			Version: modules.DataModuleVersionSourceWire{
				SourceCode: "element(\"Ca\", \"%\", \"A\")", // TODO: this shouldn't be here!
				DataModuleVersionWire: &modules.DataModuleVersionWire{
					Version:          "2.1.43",
					Tags:             []string{"oldest", "A"},
					Comments:         "Module 1",
					TimeStampUnixSec: 1234567891,
				},
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Error("Module retrieved did not match expected result")
		}
	})
}
