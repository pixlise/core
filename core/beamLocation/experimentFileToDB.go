package beamLocation

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ImportBeamLocationToDB(imgName string, instrument protos.ScanInstrument, forScanId string, beamVersion uint32, ijIndex int, fromExprPB *protos.Experiment, db *mongo.Database, logger logger.ILogger) error {
	imagesColl := db.Collection(dbCollections.ImageBeamLocationsName)

	beams := &protos.ImageLocations{
		ImageName:       imgName,
		LocationPerScan: []*protos.ImageLocationsForScan{},
	}

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

	beams.LocationPerScan = append(beams.LocationPerScan, &protos.ImageLocationsForScan{
		ScanId:      forScanId,
		Instrument:  instrument,
		BeamVersion: beamVersion,
		Locations:   ijs,
	})

	opt := options.Replace().SetUpsert(true)
	result, err := imagesColl.ReplaceOne(context.TODO(), bson.M{"_id": imgName}, beams, opt)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 && result.ModifiedCount == 0 && result.UpsertedCount == 0 {
		logger.Errorf("Image beam location insert for %v returned unexpected result %+v", imgName, result)
	}

	return nil
}
