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
		_, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](false, scanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
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

func HandleImageDeleteReq(req *protos.ImageDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ImageDeleteResp, error) {
	return nil, errors.New("HandleImageDeleteReq not implemented yet")
}
func HandleImageSetDefaultReq(req *protos.ImageSetDefaultReq, hctx wsHelpers.HandlerContext) (*protos.ImageSetDefaultResp, error) {
	return nil, errors.New("HandleImageSetDefaultReq not implemented yet")
}
func HandleImageUploadReq(req *protos.ImageUploadReq, hctx wsHelpers.HandlerContext) (*protos.ImageUploadResp, error) {
	return nil, errors.New("HandleImageUploadReq not implemented yet")
}
