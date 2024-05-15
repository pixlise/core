package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleWidgetDataGetReq(req *protos.WidgetDataGetReq, hctx wsHelpers.HandlerContext) ([]*protos.WidgetDataGetResp, error) {
	result := hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).FindOne(context.TODO(), bson.M{
		"_id": req.Id,
	})

	if result.Err() != nil {
		return nil, result.Err()
	}

	widgetData := &protos.WidgetData{}
	err := result.Decode(&widgetData)
	if err != nil {
		return nil, err
	}

	return []*protos.WidgetDataGetResp{&protos.WidgetDataGetResp{
		WidgetData: widgetData,
	}}, nil
}

func HandleWidgetDataWriteReq(req *protos.WidgetDataWriteReq, hctx wsHelpers.HandlerContext) ([]*protos.WidgetDataWriteResp, error) {
	if req.WidgetData.Id == "" {
		return nil, errors.New("widget data must have a predefined id to write to")
	}

	// Check if exists
	result := hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).FindOne(context.TODO(), bson.M{
		"_id": req.WidgetData.Id,
	})

	// The widget must already exist for us to write to it (created by screen configuration)
	if result.Err() != nil {
		return nil, result.Err()
	}

	// Update
	_, err := hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).UpdateOne(context.TODO(), bson.M{
		"_id": req.WidgetData.Id,
	}, bson.M{
		"$set": req.WidgetData,
	})

	if err != nil {
		return nil, err
	}

	return []*protos.WidgetDataWriteResp{&protos.WidgetDataWriteResp{
		WidgetData: req.WidgetData,
	}}, nil
}
