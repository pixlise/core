package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandleImageBeamLocationsReq(req *protos.ImageBeamLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ImageBeamLocationsResp, error) {
	if err := wsHelpers.CheckStringField(&req.ImageName, "ImageName", 1, 255); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

	// Read the image and check that the user has access to all scans associated with it
	result := coll.FindOne(ctx, bson.M{"_id": req.ImageName})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.ImageName)
		}
		return nil, result.Err()
	}

	locs := protos.ImageLocations{}
	err := result.Decode(&locs)
	if err != nil {
		return nil, err
	}

	if len(locs.LocationPerScan) <= 0 {
		return nil, fmt.Errorf("No beams defined for image: %v", req.ImageName)
	}

	for _, scanLocs := range locs.LocationPerScan {
		_, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](false, scanLocs.ScanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
		if err != nil {
			return nil, err
		}
	}

	// Return the coordinates from DB record
	return &protos.ImageBeamLocationsResp{
		Locations: &locs,
	}, nil
}
