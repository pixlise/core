package wsHelpers

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SessionUser struct {
	SessionId              string
	User                   *protos.UserInfo
	Permissions            map[string]bool
	MemberOfGroupIds       []string
	ViewerOfGroupIds       []string
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
var cachedUserGroupViewership = map[string][]string{}

// JWT user has the user ID and permissions that we get from Auth0. The rest is handled
// within PIXLISE, so lets read our DB to see if this user exists and get their
// user name, email, icon, etc
func MakeSessionUser(sessionId string, userId string, permissions map[string]bool, db *mongo.Database) (*SessionUser, error) {
	// Ensure we have the full user ID, as our system was previously cutting the prefix
	// off of Auth0 user ids
	fixedUserId := utils.FixUserId(userId)

	userDBItem, err := GetDBUser(fixedUserId, db)
	if err != nil {
		return nil, err
	}

	return makeSessionUser(fixedUserId, sessionId, permissions, userDBItem, db)
}

// If we have a successful login and the user is not in our DB, we write a default record
// for them, so if they change their details we have a spot to save it already
// NOTE: Non-session users can also be created via CreateNonSessionDBUser in userDBCache.go
func CreateDBUser(sessionId string, jwtUser jwtparser.JWTUserInfo, db *mongo.Database, defaultGroupIdToJoin string, log logger.ILogger) (*SessionUser, error) {
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

	ctx := context.TODO()
	_, err := db.Collection(dbCollections.UsersName).InsertOne(ctx, userDBItem)
	if err != nil {
		return nil, err
	}

	// Auto-insert into a group if configured
	if len(defaultGroupIdToJoin) > 0 {
		result, err := db.Collection(dbCollections.UserGroupsName).UpdateByID(ctx, defaultGroupIdToJoin, bson.D{{Key: "$addToSet", Value: bson.D{{Key: "members.userids", Value: userId}}}})
		if err != nil {
			log.Errorf("Failed to add user %v to default group %v. Error: %v", userId, defaultGroupIdToJoin, err)
		} else if result.MatchedCount != 1 {
			log.Infof("Unexpected update count when adding user %v to default group %v: %+v", userId, defaultGroupIdToJoin, result)
		} else {
			log.Infof("New user %v added to default group: %v", userId, defaultGroupIdToJoin)
		}
	}

	return makeSessionUser(userId, sessionId, jwtUser.Permissions, userDBItem, db)
}

func makeSessionUser(userId string, sessionId string, permissions map[string]bool, userDBItem *protos.UserDBItem, db *mongo.Database) (*SessionUser, error) {
	ourGroups := map[string]bool{} // Map of group IDs we are members of - true for members, false for viewers

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
		}

		if _, userInGroup := ourGroups[userGroup.Id]; userGroup.Viewers != nil && !userInGroup {
			if utils.ItemInSlice(userId, userGroup.Viewers.UserIds) {
				ourGroups[userGroup.Id] = false
			}
		}
	}

	// Finally, if we are in a group which itself is also within a group, find again
	// TODO: This may not detect outside of 2 levels deep grouping, we may want more...
	for _, userGroup := range userGroups {
		if userGroup.Members == nil && userGroup.Viewers == nil {
			continue
		}

		// If we are already a member of this group, we don't need to check for additional permissions
		if _, isMemberOfGroup := ourGroups[userGroup.Id]; isMemberOfGroup {
			continue
		}

		// Check if any group we're a member of is a member of this group
		for groupToCheck, isMemberOfGroupToCheck := range ourGroups {
			if userGroup.Id != groupToCheck {
				if userGroup.Members != nil {
					// If a group we're in is a member of this group, we have the same permissions (eg. viewer of a member group is still a viewer)
					if utils.ItemInSlice(groupToCheck, userGroup.Members.GroupIds) {
						ourGroups[userGroup.Id] = isMemberOfGroupToCheck
					}
				}

				if _, userInGroup := ourGroups[userGroup.Id]; userGroup.Viewers != nil && !userInGroup {
					if utils.ItemInSlice(groupToCheck, userGroup.Viewers.GroupIds) {
						ourGroups[userGroup.Id] = false
					}
				}
			}
		}
	}

	memberOfGroups := []string{}
	viewerOfGroups := []string{}

	for item, isMember := range ourGroups {
		if isMember {
			memberOfGroups = append(memberOfGroups, item)
		} else {
			viewerOfGroups = append(viewerOfGroups, item)
		}
	}

	// Any time we create a session user, we cache the list of groups it's a member of
	// so that HTTP endpoints can also access this and determine permissions properly
	cachedUserGroupMembership[userId] = memberOfGroups
	cachedUserGroupViewership[userId] = viewerOfGroups

	return &SessionUser{
		SessionId:        sessionId,
		User:             userDBItem.Info,
		Permissions:      permissions,
		MemberOfGroupIds: memberOfGroups,
		ViewerOfGroupIds: viewerOfGroups,
	}, nil
}

func GetCachedUserGroupMembership(userId string) ([]string, bool) {
	membership, ok := cachedUserGroupMembership[userId]
	return membership, ok
}

func GetCachedUserGroupViewership(userId string) ([]string, bool) {
	membership, ok := cachedUserGroupViewership[userId]
	return membership, ok
}

type UserImpersonationItem struct {
	// The user id who is doing the impersonating
	Id string `bson:"_id"`

	// The id of the user we're pretending to be
	ImpersonatedId string

	// For ease of debugging, we don't "time it out" right now or anything...
	TimeStampUnixSec uint32
}
