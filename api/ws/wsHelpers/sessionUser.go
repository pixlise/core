package wsHelpers

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/jwtparser"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SessionUser struct {
	SessionId              string
	User                   *protos.UserInfo
	Permissions            map[string]bool
	MemberOfGroupIds       []string
	NotificationSubscribed bool
}

func GetSessionUser(s *melody.Session) (SessionUser, error) {
	var connectingUser SessionUser

	if _connectingUser, ok := s.Get("user"); !ok {
		return connectingUser, errors.New("User not found on session")
	} else {
		connectingUser, ok = _connectingUser.(SessionUser)
		if !ok {
			return connectingUser, errors.New("User details corrupt on session")
		}
	}

	return connectingUser, nil
}

var cachedUserGroupMembership = map[string][]string{}

// JWT user has the user ID and permissions that we get from Auth0. The rest is handled
// within PIXLISE, so lets read our DB to see if this user exists and get their
// user name, email, icon, etc
func MakeSessionUser(sessionId string, jwtUser jwtparser.JWTUserInfo, db *mongo.Database) (*SessionUser, error) {
	// Ensure we have the full user ID, as our system was previously cutting the prefix
	// off of Auth0 user ids
	userId := utils.FixUserId(jwtUser.UserID)

	userDBItem, err := GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	return makeSessionUser(userId, sessionId, jwtUser.Permissions, userDBItem, db)
}

// If we have a successful login and the user is not in our DB, we write a default record
// for them, so if they change their details we have a spot to save it already
// NOTE: This is (at time of writing) the only way to add a user to the DB
func CreateDBUser(sessionId string, jwtUser jwtparser.JWTUserInfo, db *mongo.Database) (*SessionUser, error) {
	userId := utils.FixUserId(jwtUser.UserID)

	userDBItem := &protos.UserDBItem{
		Id: userId,
		Info: &protos.UserInfo{
			Id:    userId,
			Name:  jwtUser.Name,
			Email: jwtUser.Email,
			// IconURL
		},
		DataCollectionVersion: "",
		//NotificationSettings
	}

	_, err := db.Collection(dbCollections.UsersName).InsertOne(context.TODO(), userDBItem)
	if err != nil {
		return nil, err
	}

	// TODO: do we insert it in any groups?

	return makeSessionUser(userId, sessionId, jwtUser.Permissions, userDBItem, db)
}

func makeSessionUser(userId string, sessionId string, permissions map[string]bool, userDBItem *protos.UserDBItem, db *mongo.Database) (*SessionUser, error) {
	ourGroups := map[string]bool{}

	// Now we read all the groups and find which ones we are members of
	filter := bson.D{}
	opts := options.Find()
	cursor, err := db.Collection(dbCollections.UserGroupsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	userGroups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &userGroups)
	if err != nil {
		return nil, err
	}

	for _, userGroup := range userGroups {
		if userGroup.Members != nil {
			if utils.ItemInSlice(userId, userGroup.Members.UserIds) {
				ourGroups[userGroup.Id] = true
			}
		} else if userGroup.Viewers != nil {
			if utils.ItemInSlice(userId, userGroup.Viewers.UserIds) {
				ourGroups[userGroup.Id] = true
			}
		}
	}

	// Finally, if we are in a group which itself is also within a group, find again
	// TODO: This may not detect outside of 2 levels deep grouping, we may want more...
	for _, userGroup := range userGroups {
		for groupToCheck, _ := range ourGroups {
			if userGroup.Id != groupToCheck {
				if userGroup.Members != nil {
					if utils.ItemInSlice(groupToCheck, userGroup.Members.GroupIds) {
						ourGroups[userGroup.Id] = true
					}
				} else if userGroup.Viewers != nil {
					if utils.ItemInSlice(groupToCheck, userGroup.Viewers.GroupIds) {
						ourGroups[userGroup.Id] = true
					}
				}
			}
		}
	}

	memberOfGroups := utils.GetMapKeys(ourGroups)

	// Any time we create a session user, we cache the list of groups it's a member of
	// so that HTTP endpoints can also access this and determine permissions properly
	cachedUserGroupMembership[userId] = memberOfGroups

	return &SessionUser{
		SessionId:        sessionId,
		User:             userDBItem.Info,
		Permissions:      permissions,
		MemberOfGroupIds: memberOfGroups,
	}, nil
}

func GetCachedUserGroupMembership(userId string) ([]string, bool) {
	membership, ok := cachedUserGroupMembership[userId]
	return membership, ok
}