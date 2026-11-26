package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/memoisation"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleExpressionOutputReq(req *protos.ExpressionOutputReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionOutputResp, error) {
	if req.Request == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Expected request item"))
	}

	if hctx.SessUser.User.Id == "" {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("User must be logged in"))
	}

	reqItem := req.Request

	// Sanity check
	if err := wsHelpers.CheckStringField(&reqItem.ScanId, "scanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&reqItem.QuantId, "quantId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&reqItem.ExpressionId, "expressionId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&reqItem.RoiId, "roiId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	scanItem, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](true, reqItem.ScanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to read scan item for scan \"%v\": %v", reqItem.ScanId, err)
	}

	exprItem, _, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, reqItem.ExpressionId, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to read expression for scan \"%v\": %v", reqItem.ExpressionId, err)
	}

	// Work out what key this data would be stored under
	cacheKey, err := memoisation.MakeCacheKey(scanItem, exprItem, reqItem.QuantId, reqItem.RoiId, reqItem.Units)
	if err != nil {
		return nil, err
	}

	// We're pessimistic - assume data is not available yet...
	jobIdToWaitFor := ""

	// Check if we have anything saved in memoisation for this key
	filter := bson.M{"_id": cacheKey}
	opts := options.FindOne().SetProjection(bson.D{{Key: "id", Value: true}})
	result := hctx.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), filter, opts)
	if result.Err() != nil {
		if result.Err() != mongo.ErrNoDocuments {
			// We got some other error when trying to read the expression result, stop here
			return nil, result.Err()
		}

		// Data is simply not found yet, so here we trigger the expression to run!
		state, err := hctx.Svcs.JobManager.SubmitExpressionJob(reqItem.ScanId, reqItem.QuantId, reqItem.ExpressionId, reqItem.RoiId, cacheKey, &hctx.SessUser, hctx.Session)
		if err != nil {
			return nil, fmt.Errorf("Expression output was not available, however the calculation job failed to start: %v", err)
		}

		// if state.Status != protos.JobStatus_STARTING {
		// 	return nil, fmt.Errorf("Expression output was not available, and calculation job has unexpected state: %v", state)
		// }
		hctx.Svcs.Log.Infof("Signalling client to wait for expression calculation with job ID: %v", state.JobId)

		// Return the key that the client will have to listen for to know when to
		// retrieve the data calculated
		jobIdToWaitFor = state.JobId
	}

	return &protos.ExpressionOutputResp{
		Key:   cacheKey,
		JobId: jobIdToWaitFor,
	}, nil
}

/*
	func readExpressionResult(memCacheKey string, hctx wsHelpers.HandlerContext) (*protos.RegionDataResultItem, error) {
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
			return nil, fmt.Errorf("Failed to read memoised item for scan \"%v\": %v", memCacheKey, err)
		}

		agePastMax := memoItemAgePastMax(memItem, hctx.Svcs)
		if agePastMax > 0 {
			hctx.Svcs.Log.Infof("Memoised item: \"%v\" is %v sec too old. Not using for expression result", memCacheKey, agePastMax)
			return nil, mongo.ErrNoDocuments
		}

		// Decode its embedded data
		memResult, err := fromMemoised(memItem.Data)
		if err != nil {
			return nil, fmt.Errorf("Failed to read memoised data for scan \"%v\": %v", memCacheKey, err)
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
*/
