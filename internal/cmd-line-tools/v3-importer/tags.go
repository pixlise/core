package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type SrcTagID struct {
	ID string `json:"id"`
}

type SrcTag struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Creator     SrcUserInfo `json:"creator"`
	DateCreated int64       `json:"dateCreated"`
	Type        string      `json:"type"`
	DatasetID   string      `json:"datasetID"`
}

type SrcTags []SrcTag

type SrcTagLookup map[string]SrcTag

func migrateTags(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.TagsName)

	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	destTags := []interface{}{}
	allItems := SrcTagLookup{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, "Tags.json") {
			// Read this file
			items := SrcTagLookup{}
			err = fs.ReadJSON(userContentBucket, p, &items, false)
			if err != nil {
				return err
			}

			if !strings.HasPrefix(p, "UserContent/shared/") {
				return fmt.Errorf("Unexpected Tags.json: %v", p)
			} else {
				// Write these to DB and also remember them for later...
				for id, item := range items {
					if ex, ok := allItems[id]; ok {
						fmt.Printf("Duplicate: %v - %v vs %v\n", id, item.Name, ex.Name)
						continue
					}
					allItems[id] = item

					destTag := protos.TagDB{
						Id:      item.ID,
						Name:    item.Name,
						Type:    item.Type,
						ScanId:  item.DatasetID,
						OwnerId: item.Creator.UserID,
					}

					destTags = append(destTags, &destTag)
				}
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destTags)
	if err != nil {
		return err
	}

	fmt.Printf("Tags inserted: %v\n", len(result.InsertedIDs))

	return err
}
