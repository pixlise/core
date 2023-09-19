package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/indexcompression"
	"github.com/pixlise/core/v3/core/utils"
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

func migrateROIs(
	userContentBucket string,
	userContentFiles []string,
	limitToDatasetIds []string,
	fs fileaccess.FileAccess,
	dest *mongo.Database,
	userGroups map[string]string) error {
	coll := dest.Collection(dbCollections.RegionsOfInterestName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	sharedItems := SrcROILookup{}
	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ROI.json") && strings.HasPrefix(p, "UserContent/shared/") {
			scanId := filepath.Base(filepath.Dir(p))

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(scanId, limitToDatasetIds) {
				fmt.Printf(" SKIPPING shared roi for dataset id: %v...\n", scanId)
				continue
			}

			// Read this file
			items := SrcROILookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			// Store these till we're finished here
			for id, item := range items {
				sharedItems[ /*scanId+"_"+*/ id] = item
			}
			sharedItems = items
		}
	}

	destROIs := []interface{}{}
	allItems := SrcROILookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ROI.json") && !strings.HasPrefix(p, "UserContent/shared/") {
			scanId := filepath.Base(filepath.Dir(p))
			userIdFromPath := filepath.Base(filepath.Dir(filepath.Dir(p)))

			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf(" SKIPPING import of ROI from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(scanId, limitToDatasetIds) {
				fmt.Printf(" SKIPPING roi for dataset id: %v...\n", scanId)
				continue
			}

			// Read this file
			items := SrcROILookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			// Write these to DB and also remember them for later...
			for id, item := range items {
				saveId := /*scanId + "_" +*/ id

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

				scanIdxs, err := indexcompression.EncodeIndexList(item.LocationIndexes)
				if err != nil {
					return fmt.Errorf("ROI %v: location list error: %v", saveId, err)
				}
				pixIdxs, err := indexcompression.EncodeIndexList(item.PixelIndexes)
				if err != nil {
					return fmt.Errorf("ROI %v: pixel list error: %v", saveId, err)
				}

				destROI := protos.ROIItem{
					Id:                      saveId,
					ScanId:                  scanId,
					Name:                    item.Name,
					Description:             item.Description,
					Tags:                    tags,
					ScanEntryIndexesEncoded: scanIdxs,
					ImageName:               item.ImageName,
					PixelIndexesEncoded:     pixIdxs,
					ModifiedUnixSec:         uint32(item.CreatedUnixTimeSec),
					// MistROIItem
				}

				viewerGroupId := ""
				if removeIfSharedROI(item, sharedItems) {
					viewerGroupId = userGroups["PIXL-FM"]
				}

				err = saveOwnershipItem(destROI.Id, protos.ObjectType_OT_ROI, item.Creator.UserID, "", viewerGroupId, uint32(item.CreatedUnixTimeSec), dest)
				if err != nil {
					return err
				}

				destROIs = append(destROIs, destROI)
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destROIs)
	if err != nil {
		return err
	}

	fmt.Printf("ROIs inserted: %v\n", len(result.InsertedIDs))
	fmt.Println("ROIs orphaned (shared but original not found):")
	for id := range sharedItems {
		fmt.Printf("%v\n", id)
	}

	return err
}

func removeIfSharedROI(roi SrcROISavedItem, sharedROIs SrcROILookup) bool {
	for c, sharedItem := range sharedROIs {
		if roi.Name == sharedItem.Name &&
			roi.Creator.UserID == sharedItem.Creator.UserID {
			// Remove this from the shared list
			delete(sharedROIs, c)
			return true
		}
	}

	return false
}
