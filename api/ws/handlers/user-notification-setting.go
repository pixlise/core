package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandleUserNotificationSettingsReq(req *protos.UserNotificationSettingsReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationSettingsResp, error) {
	notificationSettings, err := getUserNotificationSettingsNotNil(hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.UserNotificationSettingsResp{
		Notifications: notificationSettings,
	}, nil
}

func HandleUserNotificationSettingsWriteReq(req *protos.UserNotificationSettingsWriteReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationSettingsWriteResp, error) {
	if req.Notifications == nil || req.Notifications.TopicSettings == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Notifications must be set"))
	}

	// Lets keep it realistic in length
	if len(req.Notifications.TopicSettings) > 40 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Too many topics specified"))
	}

	// Overwrite DB field with incoming one
	userId := hctx.SessUser.User.Id
	update := bson.D{{"notificationsettings", req.Notifications}}
	_ /*result*/, err := hctx.Svcs.MongoDB.Collection(dbCollections.UsersName).UpdateByID(context.TODO(), userId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	return &protos.UserNotificationSettingsWriteResp{}, nil
}

func getUserNotificationSettingsNotNil(userId string, db *mongo.Database) (*protos.UserNotificationSettings, error) {
	userDBItem, err := wsHelpers.GetDBUser(userId, db)
	if err != nil {
		return nil, err
	}

	if userDBItem.NotificationSettings == nil {
		userDBItem.NotificationSettings = &protos.UserNotificationSettings{
			TopicSettings: map[string]protos.NotificationMethod{},
		}
	}

	return userDBItem.NotificationSettings, nil
}
