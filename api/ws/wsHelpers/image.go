package wsHelpers

import (
	"context"
	"fmt"
	"path"
	"sort"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/gdsfilename"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

// type ScanImages []*protos.ScanImage

// func (s ScanImages) Len() int      { return len(s) }
// func (s ScanImages) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
// func (s ScanImages) Less(i, j int) bool { return s[i].ImagePath < s[j].ImagePath }

func GetLatestImagesOnly(images []*protos.ScanImage) ([]*protos.ScanImage, error) {
	result := []*protos.ScanImage{}
	latestImage := map[string]*protos.ScanImage{}

	// Loop through all images, find the latest version of each, and return only that
	for _, img := range images {
		// Try to decode the file name
		meta, err := gdsfilename.ParseFileName(img.ImagePath)
		if err != nil {
			// It's not named that way, so just return it
			result = append(result, img)
		} else {
			// Clear the version out and store (or add to stored one)
			thisImageVersion, err := meta.Version()
			if err != nil {
				return result, err
			}

			meta.SetVersionStr("__")

			// Preserve the path
			if len(meta.FilePath) > 0 {
				imagePath := meta.ToString(true, false)

				// If it doesn't exist, just add it
				if existingImg, ok := latestImage[imagePath]; ok {
					// Replace it if ours has a higher version
					existingMeta, err := gdsfilename.ParseFileName(existingImg.ImagePath)

					if err != nil {
						return result, err
					}

					existingVer, err := existingMeta.Version()
					if err != nil {
						return result, err
					}

					if thisImageVersion > existingVer {
						latestImage[imagePath] = img
					}
				} else {
					// Nothing exists yet for this name, so store it
					latestImage[imagePath] = img
				}
			}
		}
	}

	// Now we return all of them, sorted
	for _, img := range latestImage {
		result = append(result, img)
	}

	//sort.Sort(ScanImages(result))
	sort.Slice(result, func(i, j int) bool {
		return result[i].ImagePath > result[j].ImagePath
	})

	return result, nil
}

func GetDBImageFilter(imageName string) bson.D {
	filter := bson.D{{"_id", imageName}}

	// If it's a GDS type file name, we want to return the latest version, so we'll need to read all versions of this file name
	meta, err := gdsfilename.ParseFileName(imageName)
	if err == nil {
		// GDS file name confirmed, get the file name with version set to anything in a regex
		root := path.Dir(imageName)
		imagePath := imageName
		if len(root) > 0 {
			meta.SetVersionStr("..")
			imagePath = meta.ToString(true, false)
		}

		filter = bson.D{{"_id", primitive.Regex{Pattern: imagePath, Options: ""}}}
	}

	return filter
}
