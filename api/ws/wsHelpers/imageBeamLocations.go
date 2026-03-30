package wsHelpers

import (
	"context"
	"fmt"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetImageBeamLocations(hctx HandlerContext, imageName string, scanBeamVersions map[string]uint32) (*protos.ImageLocations, error) {
	ctx := context.TODO()
	var locs *protos.ImageLocations

	// NOTE: in the case of "matched" images, we look up the image that's marked as the matched image!
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	imgFound := coll.FindOne(ctx, bson.M{"_id": imageName}, options.FindOne())
	if imgFound.Err() != nil {
		if imgFound.Err() == mongo.ErrNoDocuments {
			// If there are no beam locations, don't return an error, just return a message with no items in it
			return &protos.ImageLocations{ImageName: imageName}, nil
		}
		return nil, imgFound.Err()
	}

	img := protos.ScanImage{}
	err := imgFound.Decode(&img)
	if err != nil {
		return nil, err
	}
	FixScanImageFileSize(&img)

	// Read the image, and follow the matched image link if there is one
	imageForBeamRead := imageName
	if img.MatchInfo != nil && len(img.MatchInfo.BeamImageFileName) > 0 {
		imageForBeamRead = img.MatchInfo.BeamImageFileName
	}
	imageForBeamRead = dataImportHelpers.GetImageNameSansVersion(imageForBeamRead)

	coll = hctx.Svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

	// Read the image and check that the user has access to all scans associated with it
	result := coll.FindOne(ctx, bson.M{"_id": imageForBeamRead})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			// If there are no beam locations, don't return an error, just return a message with no items in it
			return &protos.ImageLocations{ImageName: imageName}, nil
		}
		return nil, result.Err()
	}

	var dbLocs *protos.ImageLocations
	err = result.Decode(&dbLocs)
	if err != nil {
		return nil, err
	}

	if len(dbLocs.LocationPerScan) <= 0 {
		return nil, fmt.Errorf("No beams defined for image: %v", imageName)
	}

	// Build list of unique scans so we don't run the object access check
	dbScanIds := map[string]bool{}
	for _, scanLocs := range dbLocs.LocationPerScan {
		dbScanIds[scanLocs.ScanId] = true
	}
	scanBeamVersionsToReturn := scanBeamVersions
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
		ImageName:       imageName, // We used to return: dbLocs.ImageName but now look up the matched image here!
		LocationPerScan: []*protos.ImageLocationsForScan{},
	}

	// Return the specified scan/beam versions
	for scan, ver := range scanBeamVersionsToReturn {
		if !dbScanIds[scan] {
			return nil, fmt.Errorf("No beams defined for image: %v and scan: %v", imageName, scan)
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
			return nil, fmt.Errorf("No beams defined for image: %v and scan: %v with version: %v", imageName, scan, ver)
		}
	}

	for scanId, _ := range scanBeamVersionsToReturn {
		_, err := CheckObjectAccess(false, scanId, protos.ObjectType_OT_SCAN, hctx)
		if err != nil {
			return nil, err
		}
	}

	return locs, nil
}
