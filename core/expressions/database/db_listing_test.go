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

	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/expressions/modules"
	"github.com/pixlise/core/v2/core/pixlUser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_Module_DB_List_None(t *testing.T) {
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

		mods, err := db.ListModules()

		if err != nil {
			t.Error(err)
		}

		if len(mods) > 0 {
			t.Errorf("Expected 0 modules, got: %v", len(mods))
		}
	})
}

func Test_Module_DB_List_MissingVersion(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				1,
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
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.modules",
				mtest.NextBatch,
				bson.D{
					{"id", "mod234"},
					{"name", "Module2"},
					{"comments", "Module 2"},
					{"origin", bson.D{
						{"shared", true},
						{"creator", bson.D{
							{"name", "Peter N"},
							{"userid", "999"},
							{"email", "peter@pixlise.org"},
						}},
						{"CreatedUnixTimeSec", 1234567892},
						{"ModifiedUnixTimeSec", 1234567893},
					}},
				},
			),
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
		db := MakeExpressionDB("unit_test", &svcs)

		svcs.Expressions = db

		mods, err := db.ListModules()

		if err == nil {
			t.Error("Expected error from listing")
		}

		if err.Error() != "No versions for module mod123" {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(mods) != 0 {
			t.Errorf("Expected 0 items, got: %v", len(mods))
		}
	})
}

func Test_Module_DB_List_ReturnsOK(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// Modules
			mtest.CreateCursorResponse(
				1,
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
				}),
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.modules",
				mtest.NextBatch,
				bson.D{
					{"id", "mod234"},
					{"name", "Module2"},
					{"comments", "Module 2"},
					{"origin", bson.D{
						{"shared", true},
						{"creator", bson.D{
							{"name", "Peter N"},
							{"userid", "999"},
							{"email", "peter@pixlise.org"},
						}},
						{"CreatedUnixTimeSec", 1234567892},
						{"ModifiedUnixTimeSec", 1234567893},
					}},
				}),
			// Module 1 versions
			mtest.CreateCursorResponse(
				1,
				"modules-unit_test.moduleVersions",
				mtest.FirstBatch,
				bson.D{
					{"moduleID", "mod123"},
					{"sourceCode", "element(\"Ca\", \"%\", \"A\")"}, // TODO: this shouldn't be here!
					{"comments", "Module 1"},
					{"version", "0.1"},
					{"tags", []string{"oldest", "A"}},
					{"TimeStampUnixSec", 1234567891},
				},
			),
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.moduleVersions",
				mtest.NextBatch,
				bson.D{
					{"moduleID", "mod123"},
					{"sourceCode", "element(\"Ca\", \"%\", \"A\")"}, // TODO: this shouldn't be here!
					{"comments", "Module 1"},
					{"version", "0.2"},
					{"tags", []string{"latest"}},
					{"TimeStampUnixSec", 1234567892},
				},
			),
			// Module 2 versions
			mtest.CreateCursorResponse(
				0,
				"modules-unit_test.moduleVersions",
				mtest.FirstBatch,
				bson.D{
					{"moduleID", "mod234"},
					{"sourceCode", "element(\"Fe\", \"%\", \"A\")"}, // TODO: this shouldn't be here!
					{"comments", "Module 2"},
					{"version", "0.1"},
					{"tags", []string{"oldest"}},
					{"TimeStampUnixSec", 1234567894},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := makeMockSvcs(nil)
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("unit_test", &svcs)

		svcs.Expressions = db

		mods, err := db.ListModules()

		if err != nil {
			t.Error(err)
		}

		// Check entire returned result
		expected := modules.DataModuleWireLookup{
			"mod123": modules.DataModuleWire{
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
				Versions: []modules.DataModuleVersionWire{
					{Version: "0.1", Tags: []string{"oldest", "A"}, Comments: "Module 1", TimeStampUnixSec: 1234567891},
					{Version: "0.2", Tags: []string{"latest"}, Comments: "Module 1", TimeStampUnixSec: 1234567892},
				},
			},
			"mod234": modules.DataModuleWire{
				DataModule: &modules.DataModule{
					ID:       "mod234",
					Name:     "Module2",
					Comments: "Module 2",
					Origin: pixlUser.APIObjectItem{
						Shared:              true,
						Creator:             pixlUser.UserInfo{Name: "Peter N", UserID: "999", Email: "peter@pixlise.org"},
						CreatedUnixTimeSec:  1234567892,
						ModifiedUnixTimeSec: 1234567893,
					},
				},
				Versions: []modules.DataModuleVersionWire{
					{Version: "0.1", Tags: []string{"oldest"}, Comments: "Module 2", TimeStampUnixSec: 1234567894},
				},
			},
		}

		if !reflect.DeepEqual(mods, expected) {
			t.Error("Module list did not match expected result")
		}
	})
}
