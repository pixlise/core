package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/indexcompression"
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

func HandleScanMetaLabelsAndTypesReq(req *protos.ScanMetaLabelsAndTypesReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaLabelsAndTypesResp, error) {
	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	// Form the list of types, we have the enums defined in a new spot separate to the experiment files
	types := []protos.ScanMetaDataType{}
	for _, t := range exprPB.MetaTypes {
		tScan := protos.ScanMetaDataType_MT_STRING
		if t == protos.Experiment_MT_INT {
			tScan = protos.ScanMetaDataType_MT_INT
		} else if t == protos.Experiment_MT_FLOAT {
			tScan = protos.ScanMetaDataType_MT_FLOAT
		}
		types = append(types, tScan)
	}

	return &protos.ScanMetaLabelsAndTypesResp{
		MetaLabels: exprPB.MetaLabels,
		MetaTypes:  types,
	}, nil
}

// Utility to call for any Req message that involves serving data out of a dataset.bin file
// scanId is mandatory, but startIdx and locCount may not exist in all requests, can be set to 0 if unused/not relevant
func beginDatasetFileReqForRange(scanId string, entryRange *protos.ScanEntryRange, hctx wsHelpers.HandlerContext) (*protos.Experiment, []uint32, error) {
	if entryRange == nil {
		return nil, []uint32{}, fmt.Errorf("no entry range specified for scan %v", scanId)
	}

	exprPB, err := beginDatasetFileReq(scanId, hctx)
	if err != nil {
		return nil, []uint32{}, err
	}

	// Decode the range
	indexes, err := indexcompression.DecodeIndexList(entryRange.Indexes, len(exprPB.Locations))
	if err != nil {
		return nil, []uint32{}, err
	}

	return exprPB, indexes, nil
}

func beginDatasetFileReq(scanId string, hctx wsHelpers.HandlerContext) (*protos.Experiment, error) {
	if err := wsHelpers.CheckStringField(&scanId, "ScanId", 1, 50); err != nil {
		return nil, err
	}

	_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	// We've come this far, we have access to the scan, so read it
	exprPB, err := wsHelpers.ReadDatasetFile(scanId, hctx.Svcs)
	if err != nil {
		return nil, err
	}

	return exprPB, nil
}

func HandleScanMetaWriteReq(req *protos.ScanMetaWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanMetaWriteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Title, "Title", 1, 100); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Description, "Description", 1, 600); err != nil {
		return nil, err
	}

	_, err := wsHelpers.CheckObjectAccess(true, req.ScanId, protos.ObjectType_OT_SCAN, hctx)
	if err != nil {
		return nil, err
	}

	// Overwrites some metadata fields to allow them to be more descriptive to users. Requires permission EDIT_SCAN
	// so only admins can do this
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName)

	update := bson.D{bson.E{Key: "title", Value: req.Title}, bson.E{Key: "description", Value: req.Description}}

	result, err := coll.UpdateByID(ctx, req.ScanId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.ScanId)
	}

	return &protos.ScanMetaWriteResp{}, nil
}

func HandleScanTriggerReImportReq(req *protos.ScanTriggerReImportReq, hctx wsHelpers.HandlerContext) (*protos.ScanTriggerReImportResp, error) {
	return nil, errors.New("HandleScanTriggerReImportReq not implemented yet")
}

func HandleScanUploadReq(req *protos.ScanUploadReq, hctx wsHelpers.HandlerContext) (*protos.ScanUploadResp, error) {
	return nil, errors.New("HandleScanUploadReq not implemented yet")
}
