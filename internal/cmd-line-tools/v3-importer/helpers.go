package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func saveOwnershipItem(objectId string, objectType protos.ObjectType, editorUserId string, editorGroupId string, viewerGroupId string, timeStampUnixSec uint32, dest *mongo.Database) error {
	editorUserId = utils.FixUserId(editorUserId)

	ownerItem := &protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatorUserId:  editorUserId,
		CreatedUnixSec: timeStampUnixSec,
		Viewers: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
		Editors: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
	}

	if len(editorUserId) > 0 {
		ownerItem.Editors.UserIds = append(ownerItem.Editors.UserIds, editorUserId)
	}
	if len(editorGroupId) > 0 {
		ownerItem.Editors.GroupIds = append(ownerItem.Editors.GroupIds, editorGroupId)
	}
	if len(viewerGroupId) > 0 {
		ownerItem.Viewers.GroupIds = append(ownerItem.Viewers.GroupIds, viewerGroupId)
	}

	result, err := dest.Collection(dbCollections.OwnershipName).InsertOne(context.TODO(), ownerItem)
	if err != nil {
		return err
	}
	if result.InsertedID != objectId {
		return fmt.Errorf("Ownership insert for object %v %v inserted different id %v", objectType, objectId, result.InsertedID)
	}

	fmt.Printf(" saving ownership: %v, type: %v, editorUser: %v, editorGroup: %v, viewerGroup: %v\n", objectId, objectType, editorUserId, editorGroupId, viewerGroupId)
	return nil
}

func makeID() string {
	return utils.RandStringBytesMaskImpr(16)
}

func s3Copy(fs fileaccess.FileAccess, srcBucket string, srcPaths []string, dstBucket string, dstPaths []string, failOnError []bool) {
	var wg sync.WaitGroup

	if len(srcPaths) != len(dstPaths) || len(srcPaths) != len(failOnError) {
		fatalError(fmt.Errorf("s3Copy inputs not the same length for srcBucket %v, dstBucket %v\n", srcBucket, dstBucket))
	}

	// Copy each in its own thread
	for c, srcPath := range srcPaths {
		wg.Add(1)
		go func(srcBucket string, srcPath string, dstBucket string, dstPath string, failOnError bool) {
			defer wg.Done()

			err := fs.CopyObject(srcBucket, srcPath, dstBucket, dstPaths[c])
			if err != nil {
				if failOnError {
					fatalError(err)
				} else {
					log.Println(err)
				}
			}

			fmt.Printf("  Copied: s3://%v/%v --> s3://%v/%v\n", srcBucket, srcPath, dstBucket, dstPath)
		}(srcBucket, srcPath, dstBucket, dstPaths[c], failOnError[c])
	}

	// Wait for all
	wg.Wait()
}
