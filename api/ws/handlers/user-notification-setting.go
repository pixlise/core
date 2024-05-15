package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleUserNotificationSettingsReq(req *protos.UserNotificationSettingsReq, hctx wsHelpers.HandlerContext) ([]*protos.UserNotificationSettingsResp, error) {
	userDBItem, err := wsHelpers.GetDBUser(hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return []*protos.UserNotificationSettingsResp{&protos.UserNotificationSettingsResp{
		Notifications: userDBItem.NotificationSettings,
	}}, nil
}

func HandleUserNotificationSettingsWriteReq(req *protos.UserNotificationSettingsWriteReq, hctx wsHelpers.HandlerContext) ([]*protos.UserNotificationSettingsWriteResp, error) {
	if req.Notifications == nil || req.Notifications.TopicSettings == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Notifications must be set"))
	}

	// Lets keep it realistic in length
	if len(req.Notifications.TopicSettings) > 40 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Too many topics specified"))
	}

	// Overwrite DB field with incoming one
	userId := hctx.SessUser.User.Id
	update := bson.D{{Key: "notificationsettings", Value: req.Notifications}}
	_ /*result*/, err := hctx.Svcs.MongoDB.Collection(dbCollections.UsersName).UpdateByID(context.TODO(), userId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	return []*protos.UserNotificationSettingsWriteResp{&protos.UserNotificationSettingsWriteResp{}}, nil
}
