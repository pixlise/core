package wsHandler

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

const userCollection = "users"

func HandleUserDetailsReq(req *protos.UserDetailsReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserDetailsResp, error) {
	user, err := wsHelpers.GetSessionUser(s)
	if err != nil {
		return nil, err
	}

	// Read from DB too
	result := svcs.MongoDB.Collection(userCollection).FindOne(context.TODO(), bson.M{"_id": "auth0|" + user.UserID})
	if result.Err() != nil {
		return nil, result.Err()
	}

	userDBItem := protos.UserDBItem{}
	err = result.Decode(&userDBItem)
	if err != nil {
		return nil, err
	}

	return &protos.UserDetailsResp{
		Details: &protos.UserDetails{
			Info:                  userDBItem.Info,
			DataCollectionVersion: userDBItem.DataCollectionVersion,
			Permissions:           utils.GetStringMapKeys(user.Permissions),
		},
	}, nil
}

func HandleUserDetailsWriteReq(req *protos.UserDetailsWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserDetailsWriteResp, error) {
	return nil, errors.New("HandleUserDetailsWriteReq not implemented yet")
}
