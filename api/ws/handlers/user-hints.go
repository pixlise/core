package wsHandler

import (
	"context"

	"github.com/pixlise/core/v3/OLDCODE/core/utils"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandleUserHintsReq(req *protos.UserHintsReq, hctx wsHelpers.HandlerContext) (*protos.UserHintsResp, error) {
	hints, err := getUserHintsNotNil(hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.UserHintsResp{
		Hints: hints,
	}, nil
}

func HandleUserHintsToggleReq(req *protos.UserHintsToggleReq, hctx wsHelpers.HandlerContext) (*protos.UserHintsToggleResp, error) {
	hints, err := getUserHintsNotNil(hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	hints.Enabled = req.Enabled

	// Write back to DB
	err = writeHints(hctx.SessUser.User.Id, hints, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.UserHintsToggleResp{}, nil
}

func HandleUserDismissHintReq(req *protos.UserDismissHintReq, hctx wsHelpers.HandlerContext) (*protos.UserDismissHintResp, error) {
	userId := hctx.SessUser.User.Id
	if err := wsHelpers.CheckStringField(&req.Hint, "Hint", 1, 30); err != nil {
		return nil, err
	}

	// Read the user & add to array
	hints, err := getUserHintsNotNil(userId, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// Don't care if duplicate, just return OK
	if !utils.StringInSlice(req.Hint, hints.DismissedHints) {
		hints.DismissedHints = append(hints.DismissedHints, req.Hint)
	}

	// Write back to DB
	err = writeHints(userId, hints, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.UserDismissHintResp{}, nil
}

func getUserHintsNotNil(userId string, db *mongo.Database) (*protos.UserHints, error) {
	userDBItem, err := wsHelpers.GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	if userDBItem.Hints == nil {
		userDBItem.Hints = &protos.UserHints{Enabled: true}
	}

	if userDBItem.Hints.DismissedHints == nil {
		userDBItem.Hints.DismissedHints = []string{}
	}

	return userDBItem.Hints, nil
}

func writeHints(userId string, hints *protos.UserHints, db *mongo.Database) error {
	update := bson.D{{"hints", hints}}
	_ /*result*/, err := db.Collection(dbCollections.UsersName).UpdateByID(context.TODO(), userId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return err
	}

	// TODO: do we need to check result.MatchedCount == 1?
	return nil
}
