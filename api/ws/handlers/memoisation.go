package wsHandler

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

/* Went unused - became HTTP msgs, leaving this temporarily

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

		ScanId:  req.ScanId,
		QuantId: req.QuantId,
		ExprId:  req.ExprId,
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
}*/

func HandleMemoiseDeleteReq(req *protos.MemoiseDeleteReq, hctx wsHelpers.HandlerContext) (*protos.MemoiseDeleteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Key, "Key", 1, 1024); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)

	result, err := coll.DeleteOne(ctx, bson.D{{Key: "_id", Value: req.Key}})
	if err != nil {
		return nil, err
	}

	if result.DeletedCount != 1 {
		hctx.Svcs.Log.Errorf("MemoiseDeleteReq for: %v got unexpected DB write result: %+v", req.Key, result)
	}

	return &protos.MemoiseDeleteResp{
		Success: result.DeletedCount == 1,
	}, nil
}

func HandleMemoiseDeleteByRegexReq(req *protos.MemoiseDeleteByRegexReq, hctx wsHelpers.HandlerContext) (*protos.MemoiseDeleteByRegexResp, error) {
	if err := wsHelpers.CheckStringField(&req.Pattern, "Pattern", 1, 1024); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)

	result, err := coll.DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$regex", Value: req.Pattern}}}})
	if err != nil {
		return nil, err
	}

	if result.DeletedCount < 1 {
		hctx.Svcs.Log.Errorf("MemoiseDeleteByRegexReq for: %v got unexpected DB write result: %+v", req.Pattern, result)
	}

	return &protos.MemoiseDeleteByRegexResp{
		NumDeleted: uint32(result.DeletedCount),
	}, nil
}
