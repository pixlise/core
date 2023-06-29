package wsHelpers

import (
	"context"
	"fmt"
	"strings"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/jwtparser"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionUser struct {
	User             *protos.UserInfo
	Permissions      map[string]bool
	MemberOfGroupIds []string
}

func GetSessionUser(s *melody.Session) (SessionUser, error) {
	var sessionID = ""
	var connectingUser SessionUser

	if _id, ok := s.Get("id"); ok {
		_idStr, ok := _id.(string)
		if ok {
			sessionID = _idStr
		}
	}

	if _connectingUser, ok := s.Get("user"); !ok {
		return connectingUser, fmt.Errorf("User not found on session %v", sessionID)
	} else {
		connectingUser, ok = _connectingUser.(SessionUser)
		if !ok {
			return connectingUser, fmt.Errorf("User details corrupt on session %v", sessionID)
		}
	}

	return connectingUser, nil
}

// JWT user has the user ID and permissions that we get from Auth0. The rest is handled
// within PIXLISE, so lets read our DB to see if this user exists and get their
// user name, email, icon, etc
func ReadUser(jwtUser jwtparser.JWTUserInfo, db *mongo.Database) (*SessionUser, error) {
	// Ensure we have the full user ID, as our system was previously cutting the prefix
	// off of Auth0 user ids
	userId := jwtUser.UserID
	if !strings.HasPrefix(userId, "auth0|") {
		userId = "auth0|" + userId
	}

	result := db.Collection(dbCollections.UsersName).FindOne(context.TODO(), bson.M{"_id": userId})
	if result.Err() != nil {
		return nil, result.Err()
	}

	userDBItem := protos.UserDBItem{}
	err := result.Decode(&userDBItem)
	if err != nil {
		return nil, err
	}

	groups := []string{}

	return &SessionUser{
		User:             userDBItem.Info,
		Permissions:      jwtUser.Permissions,
		MemberOfGroupIds: groups,
	}, nil
}
