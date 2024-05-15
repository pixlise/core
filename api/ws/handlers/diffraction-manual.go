package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleDiffractionPeakManualListReq(req *protos.DiffractionPeakManualListReq, hctx wsHelpers.HandlerContext) ([]*protos.DiffractionPeakManualListResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// TODO: Check if user has access to this scan id?

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionManualPeaksName)

	filter := bson.M{"scanid": req.ScanId}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Silent error, just return empty
			return []*protos.DiffractionPeakManualListResp{&protos.DiffractionPeakManualListResp{
				Peaks: map[string]*protos.ManualDiffractionPeak{},
			}}, nil
		}

		return nil, err
	}

	result := []*protos.ManualDiffractionPeak{}
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	resultMap := map[string]*protos.ManualDiffractionPeak{}
	for _, item := range result {
		resultMap[item.Id] = item
		item.Id = ""     // Clear it, no point doubling up info, the map key contains the id already
		item.ScanId = "" // Also no point keeping this around, it was part of the request params
	}

	return []*protos.DiffractionPeakManualListResp{&protos.DiffractionPeakManualListResp{
		Peaks: resultMap,
	}}, nil
}

// NOTE: ScanId isn't checked to see if it's a real scan upon insertion!
// NOTE2: Insert ONLY! We generate an ID and insert into DB

func HandleDiffractionPeakManualInsertReq(req *protos.DiffractionPeakManualInsertReq, hctx wsHelpers.HandlerContext) ([]*protos.DiffractionPeakManualInsertResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if req.Pmc < 0 {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid PMC: %v", req.Pmc))
	}

	randStr := utils.RandStringBytesMaskImpr(6)
	id := fmt.Sprintf("%v_%v_%v", req.ScanId, req.Pmc, randStr)

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionManualPeaksName)

	peak := protos.ManualDiffractionPeak{
		Id:             id,
		ScanId:         req.ScanId,
		Pmc:            req.Pmc,
		EnergykeV:      req.EnergykeV,
		CreatedUnixSec: uint32(hctx.Svcs.TimeStamper.GetTimeNowSec()),
		CreatorUserId:  hctx.SessUser.User.Id,
	}

	result, err := coll.InsertOne(ctx, &peak)
	if err != nil {
		return nil, err
	}

	if result.InsertedID != id {
		hctx.Svcs.Log.Errorf("Manual diffraction insertion expected InsertedID of %v, got %v", id, result.InsertedID)
	}

	return []*protos.DiffractionPeakManualInsertResp{&protos.DiffractionPeakManualInsertResp{CreatedId: id}}, nil
}

func HandleDiffractionPeakManualDeleteReq(req *protos.DiffractionPeakManualDeleteReq, hctx wsHelpers.HandlerContext) ([]*protos.DiffractionPeakManualDeleteResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, wsHelpers.IdFieldMaxLength*2+1); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionManualPeaksName)

	// Delete the given item from the map stored for given scan ID
	dbResult, err := coll.DeleteOne(ctx, bson.M{"_id": req.Id})

	if err != nil {
		return nil, err
	}

	if dbResult.DeletedCount != 1 {
		// Probably didn't exist?
		return nil, errorwithstatus.MakeNotFoundError(req.Id)
	}

	return []*protos.DiffractionPeakManualDeleteResp{&protos.DiffractionPeakManualDeleteResp{}}, nil
}
