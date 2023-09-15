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

func migrateRGBMixes(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database, userGroups map[string]string) error {
	coll := dest.Collection(dbCollections.ExpressionGroupsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	sharedItems := SrcRGBMixLookup{}
	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "RGBMixes.json") && strings.HasPrefix(p, "UserContent/shared/") {
			// Read this file
			items := SrcRGBMixLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			// Store these till we're finished here
			sharedItems = items
		}
	}

	destGroups := []interface{}{}
	allItems := SrcRGBMixLookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "RGBMixes.json") && !strings.HasPrefix(p, "UserContent/shared/") {
			userIdFromPath := filepath.Base(filepath.Dir(p))
			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf(" SKIPPING import of RGB mix from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			// Read this file
			items := SrcRGBMixLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			items = fixOldIDs(items)

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

				viewerGroupId := ""
				if removeIfSharedRGBMix(item, sharedItems) {
					viewerGroupId = userGroups["PIXL-FM"]
				}

				err = saveOwnershipItem(destGroup.Id, protos.ObjectType_OT_EXPRESSION_GROUP, item.Creator.UserID, "", viewerGroupId, uint32(item.CreatedUnixTimeSec), dest)
				if err != nil {
					return err
				}

				destGroups = append(destGroups, destGroup)
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destGroups)
	if err != nil {
		return err
	}

	fmt.Printf("Expression Groups inserted: %v\n", len(result.InsertedIDs))
	fmt.Println("Expression Groups orphaned (shared but original not found):")
	for id := range sharedItems {
		fmt.Printf("%v\n", id)
	}

	return err
}

func removeIfSharedRGBMix(rgbMix SrcRGBMix, sharedRGBMixes SrcRGBMixLookup) bool {
	for c, sharedItem := range sharedRGBMixes {
		if rgbMix.Name == sharedItem.Name &&
			rgbMix.Creator.UserID == sharedItem.Creator.UserID &&
			(rgbMix.Red.ExpressionID == sharedItem.Red.ExpressionID || rgbMix.Red.Element == sharedItem.Red.Element) &&
			(rgbMix.Green.ExpressionID == sharedItem.Green.ExpressionID || rgbMix.Green.Element == sharedItem.Green.Element) &&
			(rgbMix.Blue.ExpressionID == sharedItem.Blue.ExpressionID || rgbMix.Blue.Element == sharedItem.Blue.Element) {
			// Remove this from the shared list
			delete(sharedRGBMixes, c)
			return true
		}
	}

	return false
}
