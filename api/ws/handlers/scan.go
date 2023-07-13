package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleScanListReq(req *protos.ScanListReq, hctx wsHelpers.HandlerContext) (*protos.ScanListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(context.TODO(), &scans)
	if err != nil {
		return nil, err
	}

	return &protos.ScanListResp{
		Scans: scans,
	}, nil
}

func HandleScanMetaLabelsReq(req *protos.ScanMetaLabelsReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaLabelsResp, error) {
	return nil, errors.New("HandleScanMetaLabelsReq not implemented yet")
}

func HandleScanMetaWriteReq(req *protos.ScanMetaWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaWriteResp, error) {
	return nil, errors.New("HandleScanMetaLabelsReq not implemented yet")
}

func HandleScanTriggerReImportReq(req *protos.ScanTriggerReImportReq, hctx wsHelpers.HandlerContext) (*protos.ScanTriggerReImportResp, error) {
	return nil, errors.New("HandleScanMetaLabelsReq not implemented yet")
}

func HandleScanUploadReq(req *protos.ScanUploadReq, hctx wsHelpers.HandlerContext) (*protos.ScanUploadResp, error) {
	return nil, errors.New("HandleScanMetaLabelsReq not implemented yet")
}
