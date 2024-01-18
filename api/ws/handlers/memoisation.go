package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleMemoiseGetReq(req *protos.MemoiseGetReq, hctx wsHelpers.HandlerContext) (*protos.MemoiseGetResp, error) {
	// Read from DB, if not there, fail. We do limit key sizes though
	if err := wsHelpers.CheckStringField(&req.Key, "Key", 1, 1024); err != nil {
		return nil, err
	}

	filter := bson.M{"_id": req.Key}
	opts := options.FindOne()
	result := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), filter, opts)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Key)
		}
		return nil, result.Err()
	}

	item := &protos.MemoisedItem{}
	err := result.Decode(item)

	if err != nil {
		return nil, err
	}

	return &protos.MemoiseGetResp{
		Item: item,
	}, nil
}

func HandleMemoiseWriteReq(req *protos.MemoiseWriteReq, hctx wsHelpers.HandlerContext) (*protos.MemoiseWriteResp, error) {
	// Here we overwrite freely, but we do limit key sizes though
	if err := wsHelpers.CheckStringField(&req.Key, "Key", 1, 1024); err != nil {
		return nil, err
	}
	if len(req.Data) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Missing data field"))
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	opt := options.Update().SetUpsert(true)

	timestamp := uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	item := &protos.MemoisedItem{
		Key:             req.Key,
		MemoTimeUnixSec: timestamp,
		Data:            req.Data,
	}

	result, err := coll.UpdateByID(ctx, req.Key, bson.D{{Key: "$set", Value: item}}, opt)
	if err != nil {
		return nil, err
	}

	if result.UpsertedCount != 1 {
		hctx.Svcs.Log.Errorf("MemoiseWriteReq for: %v got unexpected DB write result: %+v", req.Key, result)
	}

	return &protos.MemoiseWriteResp{
		MemoTimeUnixSec: timestamp,
	}, nil
}
