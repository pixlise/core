package wsHelpers

import (
	"path"
	"sort"

	"github.com/pixlise/core/v4/core/gdsfilename"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func FixScanImageFileSize(img *protos.ScanImage) {
	// Check file size field
	if img.FileSize64 == 0 && img.FileSize > 0 {
		img.FileSize64 = uint64(img.FileSize)
	}
}

func GetLatestImagesOnly(images []*protos.ScanImage) ([]*protos.ScanImage, error) {
	result := []*protos.ScanImage{}
	latestImage := map[string]*protos.ScanImage{}

	// Loop through all images, find the latest version of each, and return only that
	for _, img := range images {
		FixScanImageFileSize(img)

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
			meta.SetVersionStr("[0-9][0-9]")
			imagePath = meta.ToString(true, false)
		}

		filter = bson.D{{"_id", primitive.Regex{Pattern: "^" + imagePath, Options: ""}}}
	}

	return filter
}
