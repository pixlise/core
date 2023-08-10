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

// Anyone can retrieve a quant z-stack if they have quant messaging permissions
func HandleQuantCombineListGetReq(req *protos.QuantCombineListGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantCombineListGetResp, error) {
	zId := hctx.SessUser.User.Id + "_" + req.ScanId

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.QuantificationZStacksName)

	filter := bson.M{"_id": zId}
	result := coll.FindOne(ctx, filter)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.ScanId)
		}
		return nil, result.Err()
	}

	resultItem := &protos.QuantCombineItemListDB{}
	if err := result.Decode(&resultItem); err != nil {
		return nil, err
	}

	return &protos.QuantCombineListGetResp{
		List: resultItem.List,
	}, nil
}

// Anyone can save a quant z-stack if they have quant messaging permissions
func HandleQuantCombineListWriteReq(req *protos.QuantCombineListWriteReq, hctx wsHelpers.HandlerContext) (*protos.QuantCombineListWriteResp, error) {
	zId := hctx.SessUser.User.Id + "_" + req.ScanId

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.QuantificationZStacksName)

	doc := &protos.QuantCombineItemListDB{
		Id:     zId,
		UserId: hctx.SessUser.User.Id,
		ScanId: req.ScanId,
		List:   req.List,
	}

	result, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}

	if result.InsertedID != zId {
		hctx.Svcs.Log.Errorf("MultiQuant Z-Stack insert %v inserted different id %v", zId, result.InsertedID)
	}

	return &protos.QuantCombineListWriteResp{}, nil
}

func HandleMultiQuantCompareReq(req *protos.MultiQuantCompareReq, hctx wsHelpers.HandlerContext) (*protos.MultiQuantCompareResp, error) {
	return nil, errors.New("HandleMultiQuantCompareReq not implemented yet")
}

func HandleQuantCombineReq(req *protos.QuantCombineReq, hctx wsHelpers.HandlerContext) (*protos.QuantCombineResp, error) {
	return nil, errors.New("HandleQuantCombineReq not implemented yet")
}
