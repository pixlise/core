package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleImage3DModelPointsReq(req *protos.Image3DModelPointsReq, hctx wsHelpers.HandlerContext) (*protos.Image3DModelPointsResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.Image3DPointsName)
	imgFound := coll.FindOne(ctx, bson.M{"_id": req.ImageName}, options.FindOne())
	if imgFound.Err() != nil {
		if imgFound.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("3D points not found for image: \"%v\"", req.ImageName))
		}
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to read 3d points for image \"%v\": %v", req.ImageName, imgFound.Err()))
	}

	pts := &protos.Image3DPoints{}
	err := imgFound.Decode(pts)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to decode 3d points for image \"%v\": %v", req.ImageName, err))
	}

	return &protos.Image3DModelPointsResp{
		Points: pts,
	}, nil
}

func HandleImage3DModelPointUploadReq(req *protos.Image3DModelPointUploadReq, hctx wsHelpers.HandlerContext) (*protos.Image3DModelPointUploadResp, error) {
	if req.Points == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Points is empty"))
	}

	if len(req.Points.ImageName) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Points.ImageName is not set"))
	}

	if len(req.Points.Points) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Point list is empty"))
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	imgFound := coll.FindOne(ctx, bson.M{"_id": req.Points.ImageName}, options.FindOne())
	if imgFound.Err() != nil {
		if imgFound.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Image \"%v\" not found", req.Points.ImageName))
		}
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to check image \"%v\": %v", req.Points.ImageName, imgFound.Err()))
	}

	// Request is valid, image exists, so lets store this
	coll = hctx.Svcs.MongoDB.Collection(dbCollections.Image3DPointsName)

	opt := options.Update().SetUpsert(true)

	result, err := coll.UpdateByID(ctx, req.Points.ImageName, bson.D{{Key: "$set", Value: req.Points}}, opt)
	if err != nil {
		return nil, err
	}

	if result.UpsertedCount == 0 && result.ModifiedCount == 0 {
		hctx.Svcs.Log.Errorf("HandleImage3DModelPointUploadReq got unexpected upsert result: %+v", result)
	}

	return &protos.Image3DModelPointUploadResp{}, nil
}
