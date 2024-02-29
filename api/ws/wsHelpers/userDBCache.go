package wsHelpers

import (
	"context"
	"sync"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
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

	if userDBItem.NotificationSettings == nil {
		userDBItem.NotificationSettings = &protos.UserNotificationSettings{
			TopicSettings: map[string]protos.NotificationMethod{},
		}
	}

	return &userDBItem, nil
}

// This uses a cache as it may be reading the same thing many times in bursts.
// Cache is told when user info changes, and also has a time stamp so we don't
// keep reading from cache forever
type userCacheItem struct {
	cachedInfo       *protos.UserInfo
	timestampUnixSec int64
}

var userInfoCache = map[string]userCacheItem{}
var userInfoCacheLock = sync.Mutex{}

const maxUserCacheAgeSec = 60 * 5

func getUserInfo(userId string, db *mongo.Database, ts timestamper.ITimeStamper) (*protos.UserInfo, error) {
	user := getUserInfoFromCache(userId, ts)

	if user != nil {
		return user, nil
	}

	userDBItem, err := GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	// Cache this for future
	userInfoCacheLock.Lock()
	defer userInfoCacheLock.Unlock()

	userInfoCache[userId] = userCacheItem{
		cachedInfo:       userDBItem.Info,
		timestampUnixSec: ts.GetTimeNowSec(),
	}

	return userDBItem.Info, nil
}

func getUserInfoFromCache(userId string, ts timestamper.ITimeStamper) *protos.UserInfo {
	now := ts.GetTimeNowSec()

	userInfoCacheLock.Lock()
	defer userInfoCacheLock.Unlock()

	if user, ok := userInfoCache[userId]; ok {
		// We found cached item, use if not too old
		if user.timestampUnixSec > now-maxUserCacheAgeSec {
			return user.cachedInfo
		}

		// Otherwise, do a DB read again and overwrite our cached item
	}

	return nil
}

func NotifyUserInfoChange(userId string) {
	userInfoCacheLock.Lock()
	defer userInfoCacheLock.Unlock()

	// Delete this item from our cache
	// This will ensure it is read fresh the next time this user is accessed
	delete(userInfoCache, userId)
}
