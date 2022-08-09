package notifications

import (
	"go.mongodb.org/mongo-driver/bson"
)

func toDoc(v interface{}) (doc *bson.D, err error) {
	data, err := bson.Marshal(v)
	if err != nil {
		return
	}

	err = bson.Unmarshal(data, &doc)
	return
}

//func TestGetAllMongoUsers(t *testing.T) {
//	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
//	defer mt.Close()
//	mt.Run("test name", func(mt *mtest.T) {
//		m := MongoUtils{}
//		m.userCollection = mt.Coll
//		//id1 := primitive.NewObjectID()
//		//id2 := primitive.NewObjectID()
//		//userstruct
//		u1, err := toDoc(UserStruct{
//			Userid:        "123",
//			Notifications: Notifications{},
//			Config:        Config{},
//		})
//		u2, err := toDoc(UserStruct{
//			Userid:        "456",
//			Notifications: Notifications{},
//			Config:        Config{},
//		})
//		first := mtest.CreateCursorResponse(1, "foo.bar", mtest.FirstBatch, *u1)
//		second := mtest.CreateCursorResponse(1, "foo.bar", mtest.NextBatch, *u2)
//		killCursors := mtest.CreateCursorResponse(0, "foo.bar", mtest.NextBatch)
//		mt.AddMockResponses(first, second, killCursors)
//
//		users, err := m.GetAllMongoUsers(logger.NullLogger{})
//		assert.Nil(t, err)
//		assert.Equal(t, []UserStruct{{
//			Userid:        "123",
//			Notifications: Notifications{},
//			Config:        Config{},
//		}, {
//			Userid:        "456",
//			Notifications: Notifications{},
//			Config:        Config{},
//		}}, users)
//	})
//}
