package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GenerateIJs(imageName string, scanId string, instrument protos.ScanInstrument, svcs *services.APIServices) (*protos.ImageLocations, error) {
	svcs.Log.Infof("Generating IJ's for image: \"%v\", scan: %v...", imageName, scanId)
	// Read the dataset file
	exprPB, err := ReadDatasetFile(scanId, svcs)
	if err != nil {
		return nil, err
	}

	// Generate coordinates
	scale := float32(1)
	/*
		if len(imageName) > 0 {
			scale = 100 // We scale XY up by this much to make them not be bunched up so much, so the image doesn't have to scale down too much (it's a bit arbitrary)
		}*/

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
		coll := svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

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
