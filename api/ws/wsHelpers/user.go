package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetDBUser(userId string, db *mongo.Database) (*protos.UserDBItem, error) {
	userResult := db.Collection(dbCollections.UsersName).FindOne(context.TODO(), bson.M{"_id": userId})
	if userResult.Err() != nil {
		return nil, userResult.Err()
	}

	userDBItem := protos.UserDBItem{}
	err := userResult.Decode(&userDBItem)
	if err != nil {
		return nil, err
	}

	return &userDBItem, nil
}

// This uses a cache as it may be reading the same thing many times in bursts.
// Cache is updated upon user info change though
var userInfoCache = map[string]*protos.UserInfo{}

func getUserInfo(userId string, db *mongo.Database) (*protos.UserInfo, error) {
	if user, ok := userInfoCache[userId]; ok {
		return user, nil
	}

	userDBItem, err := GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	// Cache this for future
	userInfoCache[userId] = userDBItem.Info

	return userDBItem.Info, nil
}

func NotifyUserInfoChange(userId string) {
	// Delete this item from our cache
	// This will ensure it is read fresh the next time this user is accessed
	delete(userInfoCache, userId)
}
