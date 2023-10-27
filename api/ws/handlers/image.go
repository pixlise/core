package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleImageListReq(req *protos.ImageListReq, hctx wsHelpers.HandlerContext) (*protos.ImageListResp, error) {
	if err := wsHelpers.CheckFieldLength(req.ScanIds, "ScanIds", 1, 10); err != nil {
		return nil, err
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range req.ScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	// OK all scans are accessible, so read images
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	criteria := "$in"
	if req.MustIncludeAll {
		criteria = "$all"
	}

	filter := bson.M{"associatedscanids": bson.M{criteria: req.ScanIds}}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ScanImage{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	return &protos.ImageListResp{
		Images: items,
	}, nil
}

func HandleImageGetDefaultReq(req *protos.ImageGetDefaultReq, hctx wsHelpers.HandlerContext) (*protos.ImageGetDefaultResp, error) {
	if err := wsHelpers.CheckFieldLength(req.ScanIds, "ScanIds", 1, 10); err != nil {
		return nil, err
	}

	// Check that the user has access to all the scans in question
	for _, scanId := range req.ScanIds {
		_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanDefaultImagesName)

	filter := bson.M{"_id": bson.M{"$in": req.ScanIds}}
	opts := options.Find()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ScanImageDefaultDB{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	result := map[string]string{}
	for _, item := range items {
		result[item.ScanId] = item.DefaultImageFileName
	}

	return &protos.ImageGetDefaultResp{
		DefaultImagesPerScanId: result,
	}, nil
}

func HandleImageSetDefaultReq(req *protos.ImageSetDefaultReq, hctx wsHelpers.HandlerContext) (*protos.ImageSetDefaultResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	if err := wsHelpers.CheckStringField(&req.DefaultImageFileName, "DefaultImageFileName", 1, 255); err != nil {
		return nil, err
	}

	// Write to DB
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScanDefaultImagesName)

	filter := bson.D{{"_id", req.ScanId}}
	opt := options.Update().SetUpsert(true)

	data := bson.D{
		{"$set", bson.D{
			{"defaultImageFileName", req.DefaultImageFileName},
		}},
	}

	result, err := coll.UpdateOne(ctx, filter, data, opt)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("ImageSetDefaultReq UpdateOne result had unexpected counts %+v id: %v", result, req.ScanId)
	}

	return &protos.ImageSetDefaultResp{}, nil
}

func HandleImageDeleteReq(req *protos.ImageDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ImageDeleteResp, error) {
	return nil, errors.New("HandleImageDeleteReq not implemented yet")
}

func HandleImageUploadReq(req *protos.ImageUploadReq, hctx wsHelpers.HandlerContext) (*protos.ImageUploadResp, error) {
	return nil, errors.New("HandleImageUploadReq not implemented yet")
}
