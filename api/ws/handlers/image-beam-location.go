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
)

func HandleImageBeamLocationsReq(req *protos.ImageBeamLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ImageBeamLocationsResp, error) {
	ctx := context.TODO()
	locs := protos.ImageLocations{}

	// If we have generateForScanId set, we don't want to have an image name!
	if len(req.GenerateForScanId) > 0 {
		if len(req.ImageName) > 0 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Expected empty image name for request with GenerateForScanId set"))
		}

		// Check user has access to this scan
		_, err := wsHelpers.CheckObjectAccess(false, req.GenerateForScanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}

		// Generate away! NOTE: empty image name implies this won't write to DB
		ijs, err := generateIJs("", req.GenerateForScanId, instrument, hctx)
		if err != nil {
			return nil, err
		}
	} else {
		// We MUST have an image name in this case
		if err := wsHelpers.CheckStringField(&req.ImageName, "ImageName", 1, 255); err != nil {
			return nil, err
		}

		coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

		// Read the image and check that the user has access to all scans associated with it
		result := coll.FindOne(ctx, bson.M{"_id": req.ImageName})
		if result.Err() != nil {
			if result.Err() == mongo.ErrNoDocuments {
				return nil, errorwithstatus.MakeNotFoundError(req.ImageName)
			}
			return nil, result.Err()
		}

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
	}

	// Return the coordinates from DB record
	return &protos.ImageBeamLocationsResp{
		Locations: &locs,
	}, nil
}

func generateIJs(imageName string, scanId string, instrument protos.ScanInstrument, hctx wsHelpers.HandlerContext) error {
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
			ScanId: scanId,
			//BeamVersion: 1,
			Instrument: instrument,
			Locations:  coords,
		}},
	}

	if len(imageName) > 0 {
		ctx := context.TODO()
		coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

		result, err := coll.InsertOne(ctx, &locs, options.InsertOne())
		if err != nil {
			return err
		}

		if result.InsertedID != imageName {
			return fmt.Errorf("Inserting generated beam IJs, expected id: %v, got: %v", imageName, result.InsertedID)
		}
	}

	return nil
}
