package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

type SrcROISavedItemWithDataset struct {
	DatasetId       string
	Name            string         `json:"name"`
	LocationIndexes []int32        `json:"locationIndexes"`
	Description     string         `json:"description"`
	ImageName       string         `json:"imageName,omitempty"`
	PixelIndexes    []int32        `json:"pixelIndexes,omitempty"`
	MistROIItem     SrcMistROIItem `json:"mistROIItem"`
	Tags            []string       `json:"tags"`
}
type SrcROIWithDatasetLookup map[string]SrcROISavedItemWithDataset

func migrateOrphanedSharedROIs(
	idsToMigrate []string,
	userContentBucket string,
	userContentFiles []string,
	limitToDatasetIds []string,
	fs fileaccess.FileAccess,
	dest *mongo.Database,
	viewerGroupId string) error {
	coll := dest.Collection(dbCollections.RegionsOfInterestName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ROI.json") && strings.HasPrefix(p, "UserContent/shared/") {
			scanId := filepath.Base(filepath.Dir(p))

			// Read this file
			items := SrcROILookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			// Store these till we're finished here
			for id, item := range items {
				if utils.ItemInSlice(id, idsToMigrate) {
					// Yes migrate this one
					migrateROI(id, scanId, item, coll, dest, viewerGroupId)
				}
			}
		}
	}

	return err
}

func migrateROIShares(
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

	sharedItems := SrcROIWithDatasetLookup{}
	sharedItemIds := map[string]bool{}

	roiFiles := []string{}
	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ROI.json") && !strings.Contains(p, "/BACKUP/") {
			roiFiles = append(roiFiles, p)
		}
	}

	for _, p := range roiFiles {
		if strings.HasPrefix(p, "UserContent/shared/") {
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
				if !isMistSet(item) { // ignore MIST
					if strings.HasPrefix(item.Name, "mist__roi.") {
						fmt.Println(item.Name)
					}
					sharedItems[ /*scanId+"_"+*/ id] = SrcROISavedItemWithDataset{
						DatasetId:       scanId,
						Name:            item.Name,
						LocationIndexes: item.LocationIndexes,
						Description:     item.Description,
						ImageName:       item.ImageName,
						PixelIndexes:    item.PixelIndexes,
						MistROIItem:     item.MistROIItem,
						Tags:            item.Tags,
					}
					sharedItemIds[id] = true
				}
			}
		}
	}

	shared, err := json.MarshalIndent(sharedItems, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile("roi-shared.json", shared, 0777)
	if err != nil {
		return err
	}

	allItems := SrcROIWithDatasetLookup{}
	sharedItemToNonSharedLookup := map[string]string{}
	nonSharedItemToSharedLookup := map[string]string{}

	for _, p := range roiFiles {
		if !strings.HasPrefix(p, "UserContent/shared/") {
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
				if isMistSet(item) {
					continue // ignore MIST
				}

				if ex, ok := allItems[id]; ok {
					fmt.Printf("Duplicate: %v - %v vs %v\n", id, item.Name, ex.Name)
					continue
				}

				if item.SrcAPIObjectItem.Creator.UserID != userIdFromPath {
					fmt.Printf("Unexpected ROI user: %v, path had id: %v. ROI was likely copied to another user, skipping...\n", item.SrcAPIObjectItem.Creator.UserID, userIdFromPath)
				}

				itemToSave := SrcROISavedItemWithDataset{
					DatasetId:       scanId,
					Name:            item.Name,
					LocationIndexes: item.LocationIndexes,
					Description:     item.Description,
					ImageName:       item.ImageName,
					PixelIndexes:    item.PixelIndexes,
					MistROIItem:     item.MistROIItem,
					Tags:            item.Tags,
				}
				allItems[id] = itemToSave
			}
		}
	}

	user, err := json.MarshalIndent(allItems, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile("roi-user.json", user, 0777)
	if err != nil {
		return err
	}

	for id, item := range allItems {
		sharedId := findSharedVersion(id, item, sharedItems)
		if len(sharedId) > 0 {
			sharedItemToNonSharedLookup[sharedId] = id
			nonSharedItemToSharedLookup[id] = sharedId

			delete(sharedItemIds, sharedId)
		}
	}

	// Print them out
	fmt.Println("Id->Shared Id:")
	for id, sharedId := range nonSharedItemToSharedLookup {
		fmt.Printf("%v -> %v\n", id, sharedId)
	}

	fmt.Println("Orphaned shared ids:")
	for id := range sharedItemIds {
		fmt.Printf("%v scanId: %v, name: %v\n", id, sharedItems[id].DatasetId, sharedItems[id].Name)
	}

	return err
}

func isMistSet(roi SrcROISavedItem) bool {
	return len(roi.MistROIItem.Species) > 0 || len(roi.MistROIItem.MineralGroupID) > 0 || len(roi.MistROIItem.Formula) > 0 || len(roi.MistROIItem.ClassificationTrail) > 0
}

func findSharedVersion(roiId string, roi SrcROISavedItemWithDataset, sharedROIs SrcROIWithDatasetLookup) string {
	for sharedId, sharedItem := range sharedROIs {
		// Check that they're for the same dataset id
		if roi.DatasetId == sharedItem.DatasetId {
			idxs := utils.SlicesEqual(roi.LocationIndexes, sharedItem.LocationIndexes)
			pixels := utils.SlicesEqual(roi.PixelIndexes, sharedItem.PixelIndexes)
			name := strings.Contains(roi.Name, sharedItem.Name) || strings.Contains(sharedItem.Name, roi.Name)

			compares := 0
			if idxs {
				compares++
			}
			if pixels {
				compares++
			}
			if name {
				compares++
			}

			if compares > 1 {
				fmt.Printf("user %v, shared %v, name: %v. Matches on idxs: %v, pixels: %v, name: %v\n", roiId, sharedId, roi.Name, idxs, pixels, name)
			}

			if name && idxs && pixels {
				return sharedId
			}
		}
	}

	return ""
}
