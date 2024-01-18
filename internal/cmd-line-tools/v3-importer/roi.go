package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/indexcompression"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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
	sharedItemScanIds := map[string]string{}
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
				sharedItemScanIds[id] = scanId
			}
			sharedItems = items
		}
	}

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
				if ex, ok := allItems[id]; ok {
					fmt.Printf("Duplicate: %v - %v vs %v\n", id, item.Name, ex.Name)
					continue
				}

				if item.SrcAPIObjectItem.Creator.UserID != userIdFromPath {
					fmt.Printf("Unexpected ROI user: %v, path had id: %v. ROI was likely copied to another user, skipping...\n", item.SrcAPIObjectItem.Creator.UserID, userIdFromPath)
					continue
				}

				allItems[id] = item

				viewerGroupId := ""
				if removeIfSharedROI(item, sharedItems) {
					viewerGroupId = userGroups["PIXL-FM"]
				}

				migrateROI(id, scanId, item, coll, dest, viewerGroupId)
			}
		}
	}

	fmt.Printf("ROIs inserted: %v\n", len(allItems))
	fmt.Println("Adding the following orphaned ROIs (shared but original not found):")
	for id, shared := range sharedItems {
		fmt.Printf("%v\n", id)
		migrateROI(id, sharedItemScanIds[id], shared, coll, dest, userGroups["PIXL-FM"])
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

func migrateROI(roiId string, scanId string, item SrcROISavedItem, coll *mongo.Collection, dest *mongo.Database, viewerGroupId string) error {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}

	scanIdxs, err := indexcompression.EncodeIndexList(item.LocationIndexes)
	if err != nil {
		return fmt.Errorf("ROI %v: location list error: %v", roiId, err)
	}
	pixIdxs, err := indexcompression.EncodeIndexList(item.PixelIndexes)
	if err != nil {
		return fmt.Errorf("ROI %v: pixel list error: %v", roiId, err)
	}

	destROI := protos.ROIItem{
		Id:                      roiId,
		ScanId:                  scanId,
		Name:                    item.Name,
		Description:             item.Description,
		Tags:                    tags,
		ScanEntryIndexesEncoded: scanIdxs,
		ImageName:               item.ImageName,
		PixelIndexesEncoded:     pixIdxs,
		ModifiedUnixSec:         uint32(item.CreatedUnixTimeSec),
		IsMIST:                  item.MistROIItem.ClassificationTrail != "",
	}

	_, err = coll.InsertOne(context.TODO(), &destROI)
	if err != nil {
		return err
	}

	err = saveOwnershipItem(destROI.Id, protos.ObjectType_OT_ROI, item.Creator.UserID, "", viewerGroupId, uint32(item.CreatedUnixTimeSec), dest)
	if err != nil {
		return err
	}

	if item.MistROIItem.ClassificationTrail != "" {
		err = migrateMistROI(roiId, scanId, item.MistROIItem, dest)
		if err != nil {
			return err
		}
	}

	return nil
}

func migrateMistROI(roiId string, scanId string, mistROI SrcMistROIItem, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.MistROIsName)

	destMistROI := protos.MistROIItem{
		Id:                  roiId,
		ScanId:              scanId,
		Species:             mistROI.Species,
		MineralGroupID:      mistROI.MineralGroupID,
		IdDepth:             mistROI.ID_Depth,
		ClassificationTrail: mistROI.ClassificationTrail,
		Formula:             mistROI.Formula,
	}

	_, err := coll.InsertOne(context.TODO(), &destMistROI)
	if err != nil {
		return err
	}

	return nil
}
