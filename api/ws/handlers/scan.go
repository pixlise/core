package wsHandler

import (
	"context"
	"errors"
	"fmt"

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
	exprPB, _, _, err := beginDatasetFileReq(req.ScanId, 0, 0, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.ScanMetaLabelsResp{
		MetaLabels: exprPB.MetaLabels,
	}, nil
}

// Utility to call for any Req message that involves serving data out of a dataset.bin file
// scanId is mandatory, but startIdx and locCount may not exist in all requests, can be set to 0 if unused/not relevant
func beginDatasetFileReqForRange(scanId string, entryRange *protos.ScanEntryRange, hctx wsHelpers.HandlerContext) (*protos.Experiment, uint32, uint32, error) {
	if entryRange == nil {
		return nil, 0, 0, fmt.Errorf("no entry range specified for scan %v", scanId)
	}

	return beginDatasetFileReq(scanId, entryRange.FirstEntryIndex, entryRange.EntryCount, hctx)
}

func beginDatasetFileReq(scanId string, startIdx uint32, locCount uint32, hctx wsHelpers.HandlerContext) (*protos.Experiment, uint32, uint32, error) {
	if err := wsHelpers.CheckStringField(&scanId, "ScanId", 1, 50); err != nil {
		return nil, 0, 0, err
	}

	_, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](false, scanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return nil, 0, 0, err
	}

	// We've come this far, we have access to the scan, so read it
	exprPB, err := wsHelpers.ReadDatasetFile(scanId, hctx.Svcs)
	if err != nil {
		return nil, 0, 0, err
	}

	// Check that the start index is valid
	if startIdx >= uint32(len(exprPB.Locations)) {
		return nil, 0, 0, fmt.Errorf("ScanId %v request had invalid startLocation: %v", scanId, startIdx)
	}

	// Work out the end index from request params - mainly here to standardise
	// NOTE: locCount == 0 is interpreted as ALL
	locLast := uint32(len(exprPB.Locations))
	if locCount > 0 {
		locLast = startIdx + locCount
	}

	if err != nil {
		return nil, 0, 0, err
	}

	return exprPB, startIdx, locLast, nil
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
