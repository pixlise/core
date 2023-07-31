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

type SrcChannelConfig struct {
	ExpressionID string  `json:"expressionID"`
	RangeMin     float32 `json:"rangeMin"`
	RangeMax     float32 `json:"rangeMax"`

	// We used to store this, now only here for reading in old files (backwards compatible). PIXLISE then converts it to an ExpressionID when saving again
	Element string `json:"element,omitempty"`
}

type SrcRGBMixInput struct {
	Name  string           `json:"name"`
	Red   SrcChannelConfig `json:"red"`
	Green SrcChannelConfig `json:"green"`
	Blue  SrcChannelConfig `json:"blue"`
	Tags  []string         `json:"tags"`
}
type SrcRGBMix struct {
	*SrcRGBMixInput
	*SrcAPIObjectItem
}
type SrcRGBMixLookup map[string]SrcRGBMix

func fixOldIDs(itemLookup SrcRGBMixLookup) SrcRGBMixLookup {
	// Convert any that only had an element defined to an expression ID. This is a backwards compatibility issue, we no longer store as "element"
	for _, v := range itemLookup {
		if len(v.Red.Element) > 0 && len(v.Red.ExpressionID) <= 0 {
			v.Red.ExpressionID = "expr-elem-" + v.Red.Element + "-%"
			v.Red.Element = ""
		}

		if len(v.Green.Element) > 0 && len(v.Green.ExpressionID) <= 0 {
			v.Green.ExpressionID = "expr-elem-" + v.Green.Element + "-%"
			v.Green.Element = ""
		}

		if len(v.Blue.Element) > 0 && len(v.Blue.ExpressionID) <= 0 {
			v.Blue.ExpressionID = "expr-elem-" + v.Blue.Element + "-%"
			v.Blue.Element = ""
		}
	}
	return itemLookup
}

func migrateRGBMixes(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.ExpressionGroupsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	destGroups := []interface{}{}
	allItems := SrcRGBMixLookup{}
	sharedItems := SrcRGBMixLookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "RGBMixes.json") {
			userIdFromPath := filepath.Base(filepath.Dir(p))
			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf("Skipping import of RGB mix from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			// Read this file
			items := SrcRGBMixLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			items = fixOldIDs(items)

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

					tags := item.Tags
					if tags == nil {
						tags = []string{}
					}
					destGroup := protos.ExpressionGroup{
						Id:   id,
						Name: item.Name,
						Tags: tags,
						GroupItems: []*protos.ExpressionGroupItem{
							{
								ExpressionId: item.Red.ExpressionID,
								RangeMin:     item.Red.RangeMin,
								RangeMax:     item.Red.RangeMax,
							},
							{
								ExpressionId: item.Green.ExpressionID,
								RangeMin:     item.Green.RangeMin,
								RangeMax:     item.Green.RangeMax,
							},
							{
								ExpressionId: item.Blue.ExpressionID,
								RangeMin:     item.Blue.RangeMin,
								RangeMax:     item.Blue.RangeMax,
							},
						},
					}

					err = saveOwnershipItem(destGroup.Id, protos.ObjectType_OT_EXPRESSION_GROUP, item.Creator.UserID, uint32(item.CreatedUnixTimeSec), dest)
					if err != nil {
						return err
					}

					destGroups = append(destGroups, destGroup)
				}
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destGroups)
	if err != nil {
		return err
	}

	fmt.Printf("Expression Groups inserted: %v\n", len(result.InsertedIDs))

	// Report what was shared
	for sharedId, sharedItem := range sharedItems {
		found := false
		for itemId, item := range allItems {
			if item.Name == sharedItem.Name &&
				item.Creator.UserID == sharedItem.Creator.UserID &&
				(item.Red.ExpressionID == sharedItem.Red.ExpressionID || item.Red.Element == sharedItem.Red.Element) &&
				(item.Green.ExpressionID == sharedItem.Green.ExpressionID || item.Green.Element == sharedItem.Green.Element) &&
				(item.Blue.ExpressionID == sharedItem.Blue.ExpressionID || item.Blue.Element == sharedItem.Blue.Element) {
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
