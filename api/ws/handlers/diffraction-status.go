package wsHandler

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleDiffractionPeakStatusListReq(req *protos.DiffractionPeakStatusListReq, hctx wsHelpers.HandlerContext) (*protos.DiffractionPeakStatusListResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// TODO: Check if user has access to this scan id?

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionDetectedPeakStatusesName)

	filter := bson.M{"_id": req.ScanId}
	dbResult := coll.FindOne(ctx, filter)
	if dbResult.Err() != nil {
		if dbResult.Err() == mongo.ErrNoDocuments {
			// Silent error, just return empty
			return &protos.DiffractionPeakStatusListResp{
				PeakStatuses: &protos.DetectedDiffractionPeakStatuses{
					Id:       req.ScanId,
					ScanId:   req.ScanId,
					Statuses: map[string]*protos.DetectedDiffractionPeakStatuses_PeakStatus{},
				},
			}, nil
		}
		return nil, dbResult.Err()
	}

	result := protos.DetectedDiffractionPeakStatuses{}
	err := dbResult.Decode(&result)
	if err != nil {
		return nil, err
	}

	return &protos.DiffractionPeakStatusListResp{
		PeakStatuses: &result,
	}, nil
}

// NOTE: ScanId isn't checked to see if it's a real scan upon insertion!

func HandleDiffractionPeakStatusWriteReq(req *protos.DiffractionPeakStatusWriteReq, hctx wsHelpers.HandlerContext) (*protos.DiffractionPeakStatusWriteResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.DiffractionPeakId, "DiffractionPeakId", 1, wsHelpers.IdFieldMaxLength*2+1); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionDetectedPeakStatusesName)

	// Delete the given item from the map stored for given scan ID
	item := &protos.DetectedDiffractionPeakStatuses_PeakStatus{
		Status:         req.Status,
		CreatedUnixSec: uint32(hctx.Svcs.TimeStamper.GetTimeNowSec()),
		CreatorUserId:  hctx.SessUser.User.Id,
	}

	opts := options.Update().SetUpsert(true)
	dbResult, err := coll.UpdateByID(ctx, req.ScanId, bson.D{{Key: "$set", Value: bson.M{
		"statuses." + req.DiffractionPeakId: item,
		"scanid":                            req.ScanId,
	}}}, opts)

	if err != nil {
		return nil, err
	}

	if dbResult.UpsertedCount != 1 && dbResult.ModifiedCount != 1 {
		hctx.Svcs.Log.Errorf("DiffractionPeakStatusWriteReq UpdateByID result had unexpected counts %+v", dbResult)
	}

	return &protos.DiffractionPeakStatusWriteResp{}, nil
}

func HandleDiffractionPeakStatusDeleteReq(req *protos.DiffractionPeakStatusDeleteReq, hctx wsHelpers.HandlerContext) (*protos.DiffractionPeakStatusDeleteResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.DiffractionPeakId, "DiffractionPeakId", 1, wsHelpers.IdFieldMaxLength*2+1); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DiffractionDetectedPeakStatusesName)

	// Delete the given item from the map stored for given scan ID
	dbResult, err := coll.UpdateByID(ctx, req.ScanId, bson.D{{Key: "$unset", Value: bson.D{{"statuses." + req.DiffractionPeakId, ""}}}})

	if err != nil {
		return nil, err
	}

	if dbResult.ModifiedCount != 1 {
		// Probably didn't exist?
		return nil, errorwithstatus.MakeNotFoundError(req.ScanId + "." + req.DiffractionPeakId)
		//hctx.Svcs.Log.Errorf("DiffractionPeakStatusDeleteReq UpdateByID result had unexpected counts %+v", dbResult)
	}

	return &protos.DiffractionPeakStatusDeleteResp{}, nil
}
