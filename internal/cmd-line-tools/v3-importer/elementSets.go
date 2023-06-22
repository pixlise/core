package main

import (
	"context"
	"fmt"
	"strings"

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

func migrateElementSets(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	const collectionName = "elementSets"

	err := dest.Collection(collectionName).Drop(context.TODO())
	if err != nil {
		return err
	}

	destSets := []interface{}{}
	allItems := SrcElementSetLookup{}
	sharedItems := SrcElementSetLookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "ElementSets.json") {
			// Read this file
			items := SrcElementSetLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			if strings.HasPrefix(p, "UserContent/shared/") {
				// Store these till we're finished here
				sharedItems = items
			} else {
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
						Owner: convertOwnership(*item.SrcAPIObjectItem),
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

					destSets = append(destSets, destSet)
				}
			}
		}
	}

	result, err := dest.Collection(collectionName).InsertMany(context.TODO(), destSets)
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
