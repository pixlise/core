// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package notifications

import (
	"github.com/stretchr/testify/assert"
	"github.com/pixlise/core/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
)

func toDoc(v interface{}) (doc *bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		return
	}

	err = bson.Unmarshal(data, &doc)
	return
}

func TestGetAllMongoUsers(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()
	mt.Run("test name", func(mt *mtest.T) {
		m := mongoutils{}
		m.userCollection = mt.Coll
		//id1 := primitive.NewObjectID()
		//id2 := primitive.NewObjectID()
		//userstruct
		u1, err := toDoc(UserStruct{
			Userid:        "123",
			Notifications: Notifications{},
			Config:        Config{},
		})
		u2, err := toDoc(UserStruct{
			Userid:        "456",
			Notifications: Notifications{},
			Config:        Config{},
		})
		first := mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, *u1)
		second := mtest.CreateCursorResponse(1, "foo.bar", mtest.NextBatch, *u2)
		killCursors := mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch)
		mt.AddMockResponses(first, second, killCursors)

		users, err := m.GetAllMongoUsers(logger.NullLogger{})
		assert.Nil(t, err)
		assert.Equal(t, []UserStruct{{
			Userid:        "123",
			Notifications: Notifications{},
			Config:        Config{},
		}, {
			Userid:        "456",
			Notifications: Notifications{},
			Config:        Config{},
		}}, users)
	})
}
