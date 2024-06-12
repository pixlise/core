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
	var locs *protos.ImageLocations

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

		// Read the scan item so we get the right instrument
		coll := hctx.Svcs.MongoDB.Collection(dbCollections.ScansName)
		scanResult := coll.FindOne(ctx, bson.M{"_id": req.GenerateForScanId}, options.FindOne())
		if scanResult.Err() != nil {
			return nil, errorwithstatus.MakeNotFoundError(req.GenerateForScanId)
		}

		scan := &protos.ScanItem{}
		err = scanResult.Decode(scan)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode scan: %v. Error: %v", req.GenerateForScanId, err)
		}

		// Generate away! NOTE: empty image name implies this won't write to DB
		locs, err = generateIJs("", req.GenerateForScanId, scan.Instrument, hctx)
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
				// If there are no beam locations, don't return an error, just return a message with no items in it
				return &protos.ImageBeamLocationsResp{
					Locations: &protos.ImageLocations{ImageName: req.ImageName},
				}, nil
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
		Locations: locs,
	}, nil
}

func generateIJs(imageName string, scanId string, instrument protos.ScanInstrument, hctx wsHelpers.HandlerContext) (*protos.ImageLocations, error) {
	hctx.Svcs.Log.Infof("Generating IJ's for image: %v, scan: %v...", imageName, scanId)
	// Read the dataset file
	exprPB, err := wsHelpers.ReadDatasetFile(scanId, hctx.Svcs)
	if err != nil {
		return nil, err
	}

	// Generate coordinates
	scale := float32(1)

	if len(imageName) > 0 {
		scale = 100 // We scale XY up by this much to make them not be bunched up so much, so the image doesn't have to scale down too much (it's a bit arbitrary)
	}

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
			return nil, err
		}

		if result.InsertedID != imageName {
			return nil, fmt.Errorf("Inserting generated beam IJs, expected id: %v, got: %v", imageName, result.InsertedID)
		}
	}

	return &locs, nil
}
