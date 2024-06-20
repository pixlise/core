package beamLocation

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ImportBeamLocationToDB(imgName string, instrument protos.ScanInstrument, forScanId string, beamVersion uint32, ijIndex int, fromExprPB *protos.Experiment, db *mongo.Database, logger logger.ILogger) error {
	imagesColl := db.Collection(dbCollections.ImageBeamLocationsName)
	ctx := context.TODO()
	filter := bson.M{"_id": imgName}

	beams := &protos.ImageLocations{
		ImageName:       imgName,
		LocationPerScan: []*protos.ImageLocationsForScan{},
	}

	// Try to read an existing one. If there is one, we may only be importing one of the multiple versions of beam geometry into it, we can't just blindly overwrite!
	beamReadResult := imagesColl.FindOne(ctx, filter, options.FindOne())
	if beamReadResult.Err() != nil {
		if beamReadResult.Err() != mongo.ErrNoDocuments {
			// Some notable error, stop here
			return fmt.Errorf("Failed to read existing beam location for image: %v. Error: %v", imgName, beamReadResult.Err())
		} else {
			// we're happy to use the blank one above
			logger.Infof("Image %v had no prior beam locations saved. Writing imported beam version: %v", imgName, beamVersion)
		}
	} else {
		// Use the one we read
		if err := beamReadResult.Decode(beams); err != nil {
			return fmt.Errorf("Failed to decode existing beam location for image: %v. Error: %v", imgName, err)
		}

		// We've read what's already there, ensure that the version we're writing doesn't exist!
		locs := []*protos.ImageLocationsForScan{}
		for _, beamItem := range beams.LocationPerScan {
			if beamItem.BeamVersion != beamVersion {
				locs = append(locs, beamItem)
				logger.Infof("Existing beam version %v for image %v will be preserved", beamVersion, imgName)
			} else {
				logger.Infof("Detected existing beam version %v for image %v. This will be replaced by the newly imported one", beamVersion, imgName)
			}
		}

		// Set this back in the beam we read, we're ready to append our new version
		beams.LocationPerScan = locs
	}

	// Read ij's from the experiment file
	ijs := ReadIJs(ijIndex, fromExprPB)

	beams.LocationPerScan = append(beams.LocationPerScan, &protos.ImageLocationsForScan{
		ScanId:      forScanId,
		Instrument:  instrument,
		BeamVersion: beamVersion,
		Locations:   ijs,
	})

	opt := options.Replace().SetUpsert(true)
	logger.Infof("Writing beam location to DB for image: %v, and scan: %v, instrument: %v, beamVersion: %v", imgName, forScanId, instrument, beamVersion)

	result, err := imagesColl.ReplaceOne(ctx, filter, beams, opt)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 && result.ModifiedCount == 0 && result.UpsertedCount == 0 {
		logger.Errorf("Image beam location insert for %v returned unexpected result %+v", imgName, result)
	}

	return nil
}

func ReadIJs(ijIndex int, fromExprPB *protos.Experiment) []*protos.Coordinate2D {
	// Find the coordinates for this image
	ijs := []*protos.Coordinate2D{}

	for _, loc := range fromExprPB.Locations {
		var ij *protos.Coordinate2D

		if loc.Beam != nil {
			ij = &protos.Coordinate2D{}
			if ijIndex == 0 {
				ij.I = loc.Beam.ImageI
				ij.J = loc.Beam.ImageJ
			} else {
				ij.I = loc.Beam.ContextLocations[ijIndex-1].I
				ij.J = loc.Beam.ContextLocations[ijIndex-1].J
			}
		}

		ijs = append(ijs, ij)
	}

	return ijs
}
