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
	"google.golang.org/protobuf/proto"
)

func HandleExpressionCalculateReq(req *protos.ExpressionCalculateReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionCalculateResp, error) {
	if len(req.Requests) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Expected at least one request item"))
	}

	if hctx.SessUser.User.Id == "" {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("User must be logged in"))
	}

	resultItems := []*protos.RegionDataResultItem{}

	for c, reqItem := range req.Requests {
		if len(reqItem.ScanId) <= 0 {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Request item %v must have a scan ID", c))
		}
		if len(reqItem.QuantId) <= 0 {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Request item %v must have a quant ID", c))
		}
		if len(reqItem.ExpressionId) <= 0 {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Request item %v must have an expression ID", c))
		}
		if len(reqItem.RoiId) <= 0 {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Request item %v must have a region of interest ID", c))
		}

		// For now, we just find the memoised key of the latest expression version and looking it up. If it's not pre-computed we return an error
		// Keys are of the form:
		// {"scanId":"602735105","exprId":"q2ns80oc4452eldt","quantId":"quant-aqpxxfk6i05gcsy3","roiId":"AllPoints-602735105","units":0},Resp:false,exprMod:1772129285,spectra:3298,90,0
		// So we need scan summary details and the expression last modified time
		scanItem, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](true, reqItem.ScanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to read scan item for reqItem %v (%v): %v", c, reqItem.ScanId, err)
		}

		exprItem, _, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, reqItem.ExpressionId, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
		if err != nil {
			return nil, fmt.Errorf("Failed to read expression for reqItem %v (%v): %v", c, reqItem.ExpressionId, err)
		}

		normalSpectraCount := scanItem.ContentCounts["NormalSpectra"]
		dwellSpectraCount := scanItem.ContentCounts["DwellSpectra"]

		spectrumTimeStamp := 0 // Comes from SpectrumResp.timeStampUnixSec, seems to always be 0 for now??

		memCacheKey := fmt.Sprintf(
			`{"scanId":"%v","exprId":"%v","quantId":"%v","roiId":"%v","units":%v},Resp:false,exprMod:%v,spectra:%v,%v,%v`,
			reqItem.ScanId,
			reqItem.ExpressionId,
			reqItem.QuantId,
			reqItem.RoiId,
			reqItem.Units.Number()-1,
			exprItem.ModifiedUnixSec,
			normalSpectraCount,
			dwellSpectraCount,
			spectrumTimeStamp,
		)

		// Read the item from memoisation cache
		found := true
		filter := bson.M{"_id": memCacheKey}
		opts := options.FindOne()
		result := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), filter, opts)
		if result.Err() != nil {
			if result.Err() == mongo.ErrNoDocuments {
				// Don't just quit here, we can return an individual error for this one item
				found = false
				//return nil, errorwithstatus.MakeNotFoundError(memCacheKey)
			} else {
				return nil, result.Err()
			}
		}

		var resultItem *protos.RegionDataResultItem
		if found {
			// Decode the item
			memItem := &protos.MemoisedItem{}
			err = result.Decode(memItem)

			if err != nil {
				return nil, fmt.Errorf("Failed to read memoised item for reqItem %v (%v): %v", c, memCacheKey, err)
			}

			// Decode its embedded data
			memResult, err := fromMemoised(memItem.Data)
			if err != nil {
				return nil, fmt.Errorf("Failed to read memoised data for reqItem %v (%v): %v", c, memCacheKey, err)
			}

			// Now form an item we can put in the response
			resultItem = &protos.RegionDataResultItem{
				ExprResult: memResult,
				Expression: memResult.Expression,
				// Error
				// Warning
				// RegionSettings
				// Query
				IsPMCTable: memResult.IsPMCTable,
			}
		} else {
			// Send back an error for this item
			resultItem = &protos.RegionDataResultItem{
				// ExprResult
				Expression: exprItem,
				Error:      "Failed to read cached expression result",
				// Warning
				// RegionSettings
				// Query
				IsPMCTable: false,
			}
		}

		resultItems = append(resultItems, resultItem)
	}

	return &protos.ExpressionCalculateResp{
		Result: &protos.RegionDataResults{
			QueryResults: resultItems,
			Error:        "",
		},
	}, nil
}

// Written to match fromMemoised() in client code
func fromMemoised(data []byte) (*protos.MemDataQueryResult, error) {
	memResult := &protos.MemDataQueryResult{}
	err := proto.Unmarshal(data, memResult)

	if err != nil {
		return nil, err
	}

	// NOTE: we read the same MemDataQueryResult structure as we return
	return memResult, nil
}
