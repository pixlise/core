package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/indexcompression"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type SrcMistROIItem struct {
	Species             string `json:"species"`
	MineralGroupID      string `json:"mineralGroupID"`
	ID_Depth            int32  `json:"ID_Depth"`
	ClassificationTrail string `json:"ClassificationTrail"`
	Formula             string `json:"formula"`
}
type SrcROIItem struct {
	Name            string         `json:"name"`
	LocationIndexes []int32        `json:"locationIndexes"`
	Description     string         `json:"description"`
	ImageName       string         `json:"imageName,omitempty"`
	PixelIndexes    []int32        `json:"pixelIndexes,omitempty"`
	MistROIItem     SrcMistROIItem `json:"mistROIItem"`
	Tags            []string       `json:"tags"`
}
type SrcROISavedItem struct {
	*SrcROIItem
	*SrcAPIObjectItem
}
type SrcROILookup map[string]SrcROISavedItem

func migrateROIs(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.RegionsOfInterestName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	destROIs := []interface{}{}
	allItems := SrcROILookup{}
	sharedItems := SrcROILookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ROI.json") {
			scanId := filepath.Base(filepath.Dir(p))
			userIdFromPath := filepath.Base(filepath.Dir(filepath.Dir(p)))

			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf("Skipping import of ROI from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			// Read this file
			items := SrcROILookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			if strings.HasPrefix(p, "UserContent/shared/") {
				// Store these till we're finished here
				for id, item := range items {
					sharedItems[scanId+"_"+id] = item
				}
				sharedItems = items
			} else {
				// Write these to DB and also remember them for later...
				for id, item := range items {
					saveId := scanId + "_" + id

					if ex, ok := allItems[saveId]; ok {
						fmt.Printf("Duplicate: %v - %v vs %v\n", saveId, item.Name, ex.Name)
						continue
					}

					if item.SrcAPIObjectItem.Creator.UserID != userIdFromPath {
						fmt.Printf("Unexpected ROI user: %v, path had id: %v\n", item.SrcAPIObjectItem.Creator.UserID, userIdFromPath)
					}

					allItems[saveId] = item

					tags := item.Tags
					if tags == nil {
						tags = []string{}
					}

					locIdxs, err := indexcompression.EncodeIndexList(item.LocationIndexes)
					if err != nil {
						return fmt.Errorf("ROI %v: location list error: %v", saveId, err)
					}
					pixIdxs, err := indexcompression.EncodeIndexList(item.PixelIndexes)
					if err != nil {
						return fmt.Errorf("ROI %v: pixel list error: %v", saveId, err)
					}

					destROI := protos.ROIItem{
						Id:                     saveId,
						ScanId:                 scanId,
						Name:                   item.Name,
						Description:            item.Description,
						Tags:                   tags,
						LocationIndexesEncoded: locIdxs,
						ImageName:              item.ImageName,
						PixelIndexesEncoded:    pixIdxs,
						ModifiedUnixSec:        uint32(item.CreatedUnixTimeSec),
						// MistROIItem
					}

					err = saveOwnershipItem(destROI.Id, protos.ObjectType_OT_ROI, item.Creator.UserID, uint32(item.CreatedUnixTimeSec), dest)
					if err != nil {
						return err
					}

					destROIs = append(destROIs, destROI)
				}
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destROIs)
	if err != nil {
		return err
	}

	fmt.Printf("ROIs inserted: %v\n", len(result.InsertedIDs))

	// Report what was shared
	for sharedId, sharedItem := range sharedItems {
		found := false
		for itemId, item := range allItems {
			if item.Name == sharedItem.Name &&
				item.Creator.UserID == sharedItem.Creator.UserID &&
				len(sharedItem.LocationIndexes) == len(item.LocationIndexes) &&
				sharedItem.ImageName == item.ImageName &&
				len(sharedItem.PixelIndexes) == len(item.PixelIndexes) {
				fmt.Printf("User %v item %v, named %v seems shared as %v\n", item.Creator.UserID, itemId, item.Name, sharedId)
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Shared %v item is orphaned:%+v\n", sharedId, sharedItem)
		}
	}

	return err
}
