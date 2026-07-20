package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/services"
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

		cacheKey, err := makeCacheKey(c, reqItem.ScanId, reqItem.QuantId, reqItem.ExpressionId, reqItem.RoiId, reqItem.Units, hctx)
		if err != nil {
			return nil, err
		}

		resultItem, err := readExpressionResult(c, cacheKey, hctx)

		if err == mongo.ErrNoDocuments {
			// Don't just quit here, we can return an individual error for this one item
			// We don't have this computed, so calculate it!
			hctx.Svcs.Log.Infof("ExpressionCalculateReq - Running Expression: %v for scan: %v, quant: %v...", reqItem.ExpressionId, reqItem.ScanId, reqItem.QuantId)

			var m *expressionrunner.PMCDataValues
			var goMs, totalMs uint64
			m, totalMs, goMs, err = expressionrunner.RunExpression(reqItem.ExpressionId, reqItem.ScanId, reqItem.QuantId, hctx.Svcs, false, false)

			if err != nil {
				return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to run expression %v: %v", reqItem.ExpressionId, err))
			}

			hctx.Svcs.Log.Infof("Expression \"%v\" took total %vms (%vms in Go runtime)", reqItem.ExpressionId, totalMs, goMs)

			// Memoise it!
			_, memData, err := memoise(cacheKey, reqItem.ScanId, reqItem.QuantId, reqItem.ExpressionId, reqItem.RoiId, hctx.SessUser.User.Id, m, hctx)

			if err != nil {
				return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to memoise expression result for %v: %v", reqItem.ExpressionId, err))
			}

			// Return the calculated value
			resultItem = &protos.RegionDataResultItem{
				ExprResult: memData,
				Expression: memData.Expression,
				IsPMCTable: memData.IsPMCTable,
			}

			/*err = hctx.Svcs.JobManager.SubmitExpressionJob(reqItem.ScanId, reqItem.QuantId, reqItem.ExpressionId)
			if err == nil {
				// Retrieve it again
				resultItem, err = readExpressionResult(reqItem.ScanId, reqItem.QuantId, reqItem.ExpressionId, reqItem.RoiId, reqItem.Units, hctx)
			}*/
		}

		if err != nil {
			// If we only want to return an error for the item and continue...
			/*resultItem = &protos.RegionDataResultItem{
				// ExprResult
				Expression: exprItem,
				Error:      fmt.Sprintf("Failed to read cached expression result with key: %v", memCacheKey),
				// Warning
				// RegionSettings
				// Query
				IsPMCTable: false,
			}*/
			return nil, err
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

// Written to match toMemoised() in client code
func toMemoised(m *expressionrunner.PMCDataValues, expr *protos.DataExpression) (*protos.MemDataQueryResult, []byte, error) {
	// We copy to protobuf structs which then serialise to binary
	// NOTE: This is an experiment, if it works, maybe we'll switch all code to use these structs!
	memValues := []*protos.MemPMCDataValue{}

	for _, val := range m.Values {
		memValues = append(memValues, &protos.MemPMCDataValue{
			Pmc:         uint32(val.PMC),
			Value:       float32(val.Value),
			IsUndefined: val.IsUndefined,
			Label:       val.Label,
		})
	}

	//memPixelIndexSet := result.region ? Array.from(result.region.pixelIndexSet) : [];

	memResult := &protos.MemDataQueryResult{
		ResultValues: &protos.MemPMCDataValues{
			MinValue: float32(*m.ValueRange.Min),
			MaxValue: float32(*m.ValueRange.Max),
			Values:   memValues,
			IsBinary: m.IsBinary,
			Warning:  m.Warning,
		},
		IsPMCTable: true,
	}

	if expr != nil {
		memResult.Expression = expr
	}
	/*
		if (result.region) {
			memResult.region = MemRegionSettings.create({
			region: result.region.region,
			displaySettings: ROIItemDisplaySettings.create({
				colour: result.region.displaySettings.colour.asString(),
				shape: result.region.displaySettings.shape,
			}),
			pixelIndexSet: memPixelIndexSet,
			});
		}*/

	b, err := proto.Marshal(memResult)
	return memResult, b, err
}

func makeCacheKey(resultIdx int, scanId, quantId, expressionId, roiId string, units protos.DataUnit, hctx wsHelpers.HandlerContext) (string, error) {
	// Keys are of the form:
	// {"scanId":"602735105","exprId":"q2ns80oc4452eldt","quantId":"quant-aqpxxfk6i05gcsy3","roiId":"AllPoints-602735105","units":0},Resp:false,exprMod:1772129285,spectra:3298,90,0
	// So we need scan summary details and the expression last modified time
	scanItem, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](true, scanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return "", fmt.Errorf("Failed to read scan item for reqItem %v (%v): %v", resultIdx, scanId, err)
	}

	exprItem, _, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, expressionId, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return "", fmt.Errorf("Failed to read expression for reqItem %v (%v): %v", resultIdx, expressionId, err)
	}

	normalSpectraCount := scanItem.ContentCounts["NormalSpectra"]
	dwellSpectraCount := scanItem.ContentCounts["DwellSpectra"]

	spectrumTimeStamp := 0 // Comes from SpectrumResp.timeStampUnixSec, seems to always be 0 for now??

	memCacheKey := fmt.Sprintf(
		`{"scanId":"%v","exprId":"%v","quantId":"%v","roiId":"%v","units":%v},Resp:false,exprMod:%v,spectra:%v,%v,%v`,
		scanId,
		expressionId,
		quantId,
		roiId,
		units.Number(),
		exprItem.ModifiedUnixSec,
		normalSpectraCount,
		dwellSpectraCount,
		spectrumTimeStamp,
	)

	return memCacheKey, nil
}

func memoItemAgePastMax(item *protos.MemoisedItem, svcs *services.APIServices) int64 {
	nowUnixSec := svcs.TimeStamper.GetTimeNowSec()
	maxAgeUnixSec := nowUnixSec - int64(svcs.Config.MemoiseCacheTimeOutSec)

	// Get whichever is "newer"
	// LastReadTime was introduced later, so it's optional, meaning it comes as 0 if not read from DB
	itemUnixSec := item.MemoTimeUnixSec
	if item.LastReadTimeUnixSec > itemUnixSec {
		itemUnixSec = item.LastReadTimeUnixSec
	}

	ageTooOldSec := maxAgeUnixSec - int64(itemUnixSec)
	return ageTooOldSec
}

func memoise(memCacheKey string, scanId, quantId, expressionId, roiId, requestorUserId string, m *expressionrunner.PMCDataValues, hctx wsHelpers.HandlerContext) (*protos.MemoisedItem, *protos.MemDataQueryResult, error) {
	exprItem, _, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, expressionId, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to read expression: %v. Error: %v", expressionId, err)
	}

	memResult, data, err := toMemoised(m, exprItem)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to create memoise item for expression: %v. Error: %v", expressionId, err)
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	opt := options.Update().SetUpsert(true)

	timestamp := uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	item := &protos.MemoisedItem{
		Key:                 memCacheKey,
		MemoTimeUnixSec:     timestamp,
		Data:                data,
		ScanId:              scanId,
		QuantId:             quantId,
		ExprId:              expressionId,
		DataSize:            uint32(len(data)),
		LastReadTimeUnixSec: timestamp, // Right now this is the last time it was accessed. To be updated in future get calls
		MemoWriterUserId:    requestorUserId,
	}

	result, err := coll.UpdateByID(ctx, memCacheKey, bson.D{{Key: "$set", Value: item}}, opt)
	if err != nil {
		return nil, nil, err
	}

	if result.UpsertedCount == 0 && (result.MatchedCount != result.ModifiedCount) {
		hctx.Svcs.Log.Errorf("memoise expression result for: %v got unexpected DB write result: %+v", memCacheKey, result)
	}

	return item, memResult, nil
}

func readExpressionResult(resultIdx int, memCacheKey string, hctx wsHelpers.HandlerContext) (*protos.RegionDataResultItem, error) {
	// NOTE: We just find the memoised key of the latest expression version and looking it up. If it's not pre-computed we return an error

	// Read the item from memoisation cache
	filter := bson.M{"_id": memCacheKey}
	opts := options.FindOne()
	result := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), filter, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	// Decode the item
	memItem := &protos.MemoisedItem{}

	err := result.Decode(memItem)
	if err != nil {
		return nil, fmt.Errorf("Failed to read memoised item for reqItem %v (%v): %v", resultIdx, memCacheKey, err)
	}

	agePastMax := memoItemAgePastMax(memItem, hctx.Svcs)
	if agePastMax > 0 {
		hctx.Svcs.Log.Infof("Memoised item: \"%v\" is %v sec too old. Not using for expression result", memCacheKey, agePastMax)
		return nil, mongo.ErrNoDocuments
	}

	// Decode its embedded data
	memResult, err := fromMemoised(memItem.Data)
	if err != nil {
		return nil, fmt.Errorf("Failed to read memoised data for reqItem %v (%v): %v", resultIdx, memCacheKey, err)
	}

	// Now form an item we can put in the response
	return &protos.RegionDataResultItem{
		ExprResult: memResult,
		Expression: memResult.Expression,
		// Error
		// Warning
		// RegionSettings
		// Query
		IsPMCTable: memResult.IsPMCTable,
	}, nil
}
