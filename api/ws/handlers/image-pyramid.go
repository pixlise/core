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
)

func HandleImagePyramidGetReq(req *protos.ImagePyramidGetReq, hctx wsHelpers.HandlerContext) (*protos.ImagePyramidGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, 255); err != nil {
		return nil, err
	}

	// Look up the image in DB to determine scan IDs, then determine ownership
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagePyramidsName)

	pyramidResult := coll.FindOne(ctx, bson.M{"_id": req.Id})
	if pyramidResult.Err() != nil {
		if pyramidResult.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Id)
		}
		return nil, pyramidResult.Err()
	}

	pyramid := &protos.ImagePyramidDBEntry{}
	err := pyramidResult.Decode(pyramid)
	if err != nil {
		return nil, err
	}

	if pyramid.Id != req.Id {
		hctx.Svcs.Log.Errorf("Unexpected image pyramid id: %v in pyramid %v", pyramid.Id, req.Id)
	}

	return &protos.ImagePyramidGetResp{Image: pyramid.Pyramid}, nil
}

func HandleImageTileDataGetReq(req *protos.ImageTileDataGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageTileDataGetResp, error) {
	return nil, errors.New("HandleImageTileDataGetReq not implemented yet")
}

func HandleImageTileStructureGetReq(req *protos.ImageTileStructureGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageTileStructureGetResp, error) {
	return nil, errors.New("HandleImageTileStructureGetReq not implemented yet")
}
