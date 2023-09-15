package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type SrcElementLines struct {
	AtomicNumber int8 `json:"Z"` // 118 still fits! Will we break past 127 any time soon? :)
	K            bool `json:"K"`
	L            bool `json:"L"`
	M            bool `json:"M"`
	Esc          bool `json:"Esc"`
}

type SrcElementSet struct {
	Name  string            `json:"name"`
	Lines []SrcElementLines `json:"lines"`
	*SrcAPIObjectItem
}

type SrcElementSetLookup map[string]SrcElementSet

func migrateElementSets(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database, userGroups map[string]string) error {
	coll := dest.Collection(dbCollections.ElementSetsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	sharedItems := SrcElementSetLookup{}
	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ElementSets.json") && strings.HasPrefix(p, "UserContent/shared/") {
			// Read this file
			items := SrcElementSetLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			// Store these till we're finished here
			sharedItems = items
		}
	}

	destSets := []interface{}{}
	allItems := SrcElementSetLookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ElementSets.json") && !strings.HasPrefix(p, "UserContent/shared/") {
			userIdFromPath := filepath.Base(filepath.Dir(p))
			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf(" SKIPPING import of element set from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			// Read this file
			items := SrcElementSetLookup{}
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
				allItems[id] = item

				destSet := protos.ElementSet{
					Id:    id,
					Name:  item.Name,
					Lines: []*protos.ElementLine{},
				}

				for _, line := range item.Lines {
					destSet.Lines = append(destSet.Lines, &protos.ElementLine{
						Z:   int32(line.AtomicNumber),
						K:   line.K,
						L:   line.L,
						M:   line.M,
						Esc: line.Esc,
					})
				}

				viewerGroupId := ""
				if removeIfElementSet(item, sharedItems) {
					viewerGroupId = userGroups["PIXL-FM"]
				}

				err = saveOwnershipItem(destSet.Id, protos.ObjectType_OT_ELEMENT_SET, item.Creator.UserID, "", viewerGroupId, uint32(item.CreatedUnixTimeSec), dest)
				if err != nil {
					return err
				}

				destSets = append(destSets, destSet)
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destSets)
	if err != nil {
		return err
	}

	fmt.Printf("Element sets inserted: %v\n", len(result.InsertedIDs))

	// Report what was shared
	for sharedId, sharedItem := range sharedItems {
		found := false
		for itemId, item := range allItems {
			if item.Name == sharedItem.Name && item.Creator.UserID == sharedItem.Creator.UserID && len(sharedItem.Lines) == len(item.Lines) {
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

func removeIfElementSet(elementSet SrcElementSet, sharedElementSets SrcElementSetLookup) bool {
	for c, sharedItem := range sharedElementSets {
		if elementSet.Name == sharedItem.Name && elementSet.Creator.UserID == sharedItem.Creator.UserID && len(sharedItem.Lines) == len(elementSet.Lines) {
			// Remove this from the shared list
			delete(sharedElementSets, c)
			return true
		}
	}

	return false
}
