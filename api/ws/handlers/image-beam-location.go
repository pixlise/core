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
	"go.mongodb.org/mongo-driver/mongo/options"
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
		_, err := wsHelpers.CheckObjectAccess(false, scanLocs.ScanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	// Return the coordinates from DB record
	return &protos.ImageBeamLocationsResp{
		Locations: &locs,
	}, nil
}

func generateIJs(imageName string, scanId string, hctx wsHelpers.HandlerContext) error {
	hctx.Svcs.Log.Infof("Generating IJ's for image: %v, scan: %v...", imageName, scanId)
	// Read the dataset file
	exprPB, err := wsHelpers.ReadDatasetFile(scanId, hctx.Svcs)
	if err != nil {
		return err
	}

	// Generate coordinates
	scale := float32(100.0) // We scale XY up by this much to make them not be bunched up so much, so the image doesn't have to scale down too much (it's a bit arbitrary)

	coords := []*protos.Coordinate2D{}
	for _, loc := range exprPB.Locations {
		if loc.Beam == nil {
			coords = append(coords, nil)
		} else {
			coords = append(coords, &protos.Coordinate2D{I: loc.Beam.X * scale, J: loc.Beam.Y * scale})
		}
	}

	locs := protos.ImageLocations{
		ImageName: imageName,
		LocationPerScan: []*protos.ImageLocationsForScan{{
			ScanId:    scanId,
			Locations: coords,
		}},
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

	result, err := coll.InsertOne(ctx, &locs, options.InsertOne())
	if err != nil {
		return err
	}

	if result.InsertedID != imageName {
		return fmt.Errorf("Inserting generated beam IJs, expected id: %v, got: %v", imageName, result.InsertedID)
	}

	return nil
}
