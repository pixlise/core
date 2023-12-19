package beamLocation

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func ImportBeamLocationToDB(imgName string, forScanId string, ijIndex int, fromExprPB *protos.Experiment, db *mongo.Database) error {
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
		ScanId:    forScanId,
		Locations: ijs,
	})

	result, err := imagesColl.InsertOne(context.TODO(), beams)
	if err != nil {
		return err
	}

	if result.InsertedID != imgName {
		return fmt.Errorf("Image beam location insert for %v inserted different id %v", imgName, result.InsertedID)
	}

	return nil
}
