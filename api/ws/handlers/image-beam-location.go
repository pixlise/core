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
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Expected empty imageName for request with GenerateForScanId set"))
		}
		if len(req.ScanBeamVersions) > 0 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Expected empty scanBeamVersions for request with GenerateForScanId set"))
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

		var dbLocs *protos.ImageLocations
		err := result.Decode(&dbLocs)
		if err != nil {
			return nil, err
		}

		if len(dbLocs.LocationPerScan) <= 0 {
			return nil, fmt.Errorf("No beams defined for image: %v", req.ImageName)
		}

		// Build list of unique scans so we don't run the object access check
		dbScanIds := map[string]bool{}
		for _, scanLocs := range dbLocs.LocationPerScan {
			dbScanIds[scanLocs.ScanId] = true
		}
		scanBeamVersionsToReturn := req.ScanBeamVersions
		if scanBeamVersionsToReturn == nil {
			scanBeamVersionsToReturn = map[string]uint32{}
		}

		// If they didn't specify versions to return, return the latest version for each scan represented
		if len(scanBeamVersionsToReturn) <= 0 {
			for _, scanLocs := range dbLocs.LocationPerScan {
				// Add to map if it doesn't exist yet
				if ver, ok := scanBeamVersionsToReturn[scanLocs.ScanId]; !ok {
					scanBeamVersionsToReturn[scanLocs.ScanId] = scanLocs.BeamVersion
				} else {
					// Check if this beam version is larger
					if scanLocs.BeamVersion > ver {
						scanBeamVersionsToReturn[scanLocs.ScanId] = scanLocs.BeamVersion
					}
				}
			}
		}

		// Run through what we're planning to return and make sure user has access while building the result list
		locs = &protos.ImageLocations{
			ImageName:       dbLocs.ImageName,
			LocationPerScan: []*protos.ImageLocationsForScan{},
		}

		// Return the specified scan/beam versions
		for scan, ver := range scanBeamVersionsToReturn {
			if !dbScanIds[scan] {
				return nil, fmt.Errorf("No beams defined for image: %v and scan: %v", req.ImageName, scan)
			}

			// This is a valid scan choice, now make sure the version requested exists
			verFound := false
			for _, scanLocs := range dbLocs.LocationPerScan {
				if scanLocs.ScanId == scan && scanLocs.BeamVersion == ver {
					verFound = true
					locs.LocationPerScan = append(locs.LocationPerScan, scanLocs)
					break
				}
			}

			if !verFound {
				return nil, fmt.Errorf("No beams defined for image: %v and scan: %v with version: %v", req.ImageName, scan, ver)
			}
		}

		for scanId, _ := range scanBeamVersionsToReturn {
			_, err := wsHelpers.CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
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
	hctx.Svcs.Log.Infof("Generating IJ's for image: \"%v\", scan: %v...", imageName, scanId)
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

func HandleImageBeamLocationVersionsReq(req *protos.ImageBeamLocationVersionsReq, hctx wsHelpers.HandlerContext) (*protos.ImageBeamLocationVersionsResp, error) {
	ctx := context.TODO()

	if err := wsHelpers.CheckStringField(&req.ImageName, "ImageName", 1, 255); err != nil {
		return nil, err
	}

	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)
	vers := map[string]*protos.ImageBeamLocationVersionsResp_AvailableVersions{}

	// Read the image and check that the user has access to all scans associated with it
	result := coll.FindOne(ctx, bson.M{"_id": req.ImageName})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			// If there are no beam locations, don't return an error, just return a message with no items in it
			return &protos.ImageBeamLocationVersionsResp{
				BeamVersionPerScan: vers,
			}, nil
		}
		return nil, result.Err()
	}

	var locs *protos.ImageLocations
	err := result.Decode(&locs)
	if err != nil {
		return nil, err
	}

	for _, locPerScan := range locs.LocationPerScan {
		var availVersions *protos.ImageBeamLocationVersionsResp_AvailableVersions = vers[locPerScan.ScanId]
		if availVersions == nil {
			availVersions = &protos.ImageBeamLocationVersionsResp_AvailableVersions{Versions: []uint32{}}
			vers[locPerScan.ScanId] = availVersions
		}

		availVersions.Versions = append(availVersions.Versions, locPerScan.BeamVersion)
	}

	return &protos.ImageBeamLocationVersionsResp{
		BeamVersionPerScan: vers,
	}, nil
}
