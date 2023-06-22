package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func migrateDiffraction(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	err := migrateManualDiffractionPeaks(userContentBucket, userContentFiles, fs, dest)
	if err != nil {
		return err
	}
	return nil // migrateDiffractionDetectedPeakStatuses(userContentBucket, userContentFiles, fs, dest)
}

type SrcUserDiffractionPeak struct {
	PMC int32   `json:"pmc"`
	KeV float32 `json:"keV"`
}

type SrcUserDiffractionPeakFileContents struct {
	Peaks map[string]SrcUserDiffractionPeak `json:"peaks"`
}

func migrateManualDiffractionPeaks(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	const collectionName = "diffractionUserPeaks"

	err := dest.Collection(collectionName).Drop(context.TODO())
	if err != nil {
		return err
	}

	destItems := []interface{}{}
	allItems := map[string]SrcUserDiffractionPeak{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, filepaths.DiffractionPeakManualFileName) {
			if !strings.HasPrefix(p, "UserContent/shared/") {
				return fmt.Errorf("Unexpected %v: %v", filepaths.DiffractionPeakManualFileName, p)
			} else {
				scanId := filepath.Base(filepath.Dir(p))

				// Read this file
				items := SrcUserDiffractionPeakFileContents{}
				err = fs.ReadJSON(userContentBucket, p, &items, false)
				if err != nil {
					return err
				}

				for id, item := range items.Peaks {
					if ex, ok := allItems[id]; ok {
						fmt.Printf("Duplicate: %v - kev=%v pmc=%v\n", id, item.KeV, ex.PMC)
						continue
					}
					allItems[id] = item

					destItem := protos.ManualDiffractionPeak{
						Id:        fmt.Sprintf("%v_%v_%v", scanId, item.PMC, id),
						ScanId:    scanId,
						Pmc:       item.PMC,
						EnergykeV: item.KeV,
					}

					destItems = append(destItems, destItem)
				}
			}
		}
	}

	result, err := dest.Collection(collectionName).InsertMany(context.TODO(), destItems)
	if err != nil {
		return err
	}

	fmt.Printf("User Diffraction Peaks inserted: %v\n", len(result.InsertedIDs))

	return err
}
