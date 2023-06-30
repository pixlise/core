package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleUserDetailsReq(req *protos.UserDetailsReq, hctx wsHelpers.HandlerContext) (*protos.UserDetailsResp, error) {
	// Read from DB too
	result := hctx.Svcs.MongoDB.Collection(dbCollections.UsersName).FindOne(context.TODO(), bson.M{"_id": hctx.SessUser.User.Id})
	if result.Err() != nil {
		return nil, result.Err()
	}

	userDBItem := protos.UserDBItem{}
	err := result.Decode(&userDBItem)
	if err != nil {
		return nil, err
	}

	return &protos.UserDetailsResp{
		Details: &protos.UserDetails{
			Info:                  userDBItem.Info,
			DataCollectionVersion: userDBItem.DataCollectionVersion,
			Permissions:           utils.GetStringMapKeys(hctx.SessUser.Permissions),
		},
	}, nil
}

func HandleUserDetailsWriteReq(req *protos.UserDetailsWriteReq, hctx wsHelpers.HandlerContext) (*protos.UserDetailsWriteResp, error) {
	return nil, errors.New("HandleUserDetailsWriteReq not implemented yet")
}
