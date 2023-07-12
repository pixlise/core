package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/timestamper"
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
type userCacheItem struct {
	cachedInfo       *protos.UserInfo
	timestampUnixSec int64
}

var userInfoCache = map[string]userCacheItem{}

const maxUserCacheAgeSec = 60 * 5

func getUserInfo(userId string, db *mongo.Database, ts timestamper.ITimeStamper) (*protos.UserInfo, error) {
	now := ts.GetTimeNowSec()

	if user, ok := userInfoCache[userId]; ok {
		// We found cached item, use if not too old
		if user.timestampUnixSec > now-maxUserCacheAgeSec {
			return user.cachedInfo, nil
		}

		// Otherwise, do a DB read again and overwrite our cached item
	}

	userDBItem, err := GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	// Cache this for future
	userInfoCache[userId] = userCacheItem{
		cachedInfo:       userDBItem.Info,
		timestampUnixSec: ts.GetTimeNowSec(),
	}

	return userDBItem.Info, nil
}

func NotifyUserInfoChange(userId string) {
	// Delete this item from our cache
	// This will ensure it is read fresh the next time this user is accessed
	delete(userInfoCache, userId)
}
