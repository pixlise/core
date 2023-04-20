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
	"testing"

	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/expressions/modules"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/timestamper"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_Module_DB_Create(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mongoMockedResponses := []primitive.D{
			// Module creation
			mtest.CreateSuccessResponse(),
			// Version creation
			mtest.CreateSuccessResponse(),
			// TODO: There is no way currently to "verify" what was sent to the DB!
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		idGen := services.MockIDGenerator{
			IDs: []string{"mod111"},
		}

		svcs := makeMockSvcs(&idGen)
		svcs.TimeStamper = &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1234567777},
		}
		svcs.Mongo = mt.Client
		db := MakeExpressionDB("local", &svcs)

		svcs.Expressions = db

		input := modules.DataModuleInput{
			Name:       "MyModule",
			SourceCode: "element(\"Ca\", \"%\", \"A\")",
			Comments:   "My first module",
			Tags:       []string{"A tag"},
		}
		user := pixlUser.UserInfo{Name: "Peter N", UserID: "999", Email: "peter@pixlise.org"}

		_, err := db.CreateModule(input, user)

		if err != nil {
			t.Error(err)
		}
	})
}
