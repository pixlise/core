package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/quantification"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Anyone can retrieve a quant z-stack if they have quant messaging permissions
func HandleQuantCombineListGetReq(req *protos.QuantCombineListGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantCombineListGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

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
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if req.List == nil || len(req.List.RoiZStack) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("List cannot be empty"))
	}

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
	// req.ScanId is checked in beginDatasetFileReq

	if len(req.QuantIds) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Requested with 0 quant IDs"))
	}

	// If we're requesting for RemainingPoints ROI, mandate that the PMC list is not empty, otherwise it should be
	if req.ReqRoiId == "RemainingPoints" && len(req.RemainingPointsPMCs) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("No PMCs supplied for RemainingPoints ROI"))
	} else if req.ReqRoiId != "RemainingPoints" && len(req.RemainingPointsPMCs) > 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Unexpected PMCs supplied for ROI: " + req.ReqRoiId))
	}

	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	tables, err := quantification.MultiQuantCompare(req.ReqRoiId, req.RemainingPointsPMCs, req.QuantIds, exprPB, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.MultiQuantCompareResp{
		RoiId:       req.ReqRoiId,
		QuantTables: tables,
	}, nil
}

func HandleQuantCombineReq(req *protos.QuantCombineReq, hctx wsHelpers.HandlerContext) (*protos.QuantCombineResp, error) {
	// Simple validation

	// NOTE: if only asking for a summary, we don't care about name being empty
	if !req.SummaryOnly && len(req.Name) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Name cannot be empty"))
	}

	if len(req.RoiZStack) <= 1 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Must reference more than 1 ROI"))
	}

	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	multiQuantData, err := quantification.MultiQuantCombinedCSV(req.Name, req.ScanId, req.RoiZStack, exprPB, hctx)
	if err != nil {
		return nil, err
	}

	if req.SummaryOnly {
		// We return a summary instead of forming a CSV
		summary := quantification.FormMultiQuantSummary(multiQuantData.DataPerDetectorPerPMC, multiQuantData.AllColumns, multiQuantData.PMCCount)
		return &protos.QuantCombineResp{
			CombineResult: &protos.QuantCombineResp_Summary{
				Summary: summary,
			},
		}, nil
	}

	// Form a CSV
	csv := quantification.FormCombinedCSV(multiQuantData.QuantIds, multiQuantData.DataPerDetectorPerPMC, multiQuantData.AllColumns)

	quantMode := quantification.QuantModeCombinedMultiQuant
	if len(multiQuantData.Detectors) > 1 {
		quantMode = quantification.QuantModeABMultiQuant
	}

	quantId, err := quantification.ImportQuantCSV(hctx, req.ScanId, hctx.SessUser.User, csv, "combined-multi", "multi", req.Name, quantMode, req.Description)
	if err != nil {
		return nil, err
	}

	return &protos.QuantCombineResp{
		CombineResult: &protos.QuantCombineResp_JobId{
			JobId: quantId,
		},
	}, nil
}
