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

	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/expressions/modules"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_Module_DB_AddVersion_NoModule(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// Module is missing
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

		idGen := services.MockIDGenerator{
			IDs: []string{"mod123"},
		}

		svcs := makeMockSvcs(&idGen)
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		input := modules.DataModuleVersionInput{
			SourceCode: "element(\"Ca\", \"%\", \"A\")",
			Comments:   "My comment",
			Tags:       []string{"The best"},
		}
		_, err := db.AddModuleVersion("mod123", input)

		if err == nil {
			t.Error("Expected error")
		}

		if err.Error() != "Failed to add new version to non-existant module mod123. mongo: no documents in result" {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func Test_Module_DB_AddVersion_OK(t *testing.T) {
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
					{"id", "mod123"},
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
					{"version", bson.D{
						{"major", 2},
						{"minor", 1},
						{"patch", 43},
					}},
				},
			),
			// Version write success
			mtest.CreateSuccessResponse(),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		idGen := services.MockIDGenerator{
			IDs: []string{"mod123"},
		}

		svcs := makeMockSvcs(&idGen)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1234567777},
		}
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		input := modules.DataModuleVersionInput{
			SourceCode: "element(\"Ca\", \"%\", \"A\")",
			Comments:   "My comment",
			Tags:       []string{"The best"},
		}
		result, err := db.AddModuleVersion("mod123", input)

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
				SourceCode: "element(\"Ca\", \"%\", \"A\")",
				DataModuleVersionWire: &modules.DataModuleVersionWire{
					Version:          "2.1.44",
					Tags:             []string{"The best"},
					Comments:         "My comment",
					TimeStampUnixSec: 1234567777,
				},
			},
		}

		if !reflect.DeepEqual(result, expected) {
			t.Error("Module retrieved did not match expected result")
		}
	})
}
