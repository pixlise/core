package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func saveOwnershipItem(objectId string, objectType protos.ObjectType, editorUserId string, editorGroupId string, viewerGroupId string, timeStampUnixSec uint32, dest *mongo.Database) error {
	editorUserId = utils.FixUserId(editorUserId)

	ownerItem := &protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatorUserId:  editorUserId,
		CreatedUnixSec: timeStampUnixSec,
		//Viewers: ,
		Editors: &protos.UserGroupList{},
	}

	if len(editorUserId) > 0 {
		ownerItem.Editors.UserIds = []string{editorUserId}
	}
	if len(editorGroupId) > 0 {
		ownerItem.Editors.GroupIds = []string{editorGroupId}
	}
	if len(viewerGroupId) > 0 {
		ownerItem.Viewers = &protos.UserGroupList{GroupIds: []string{viewerGroupId}}
	}

	result, err := dest.Collection(dbCollections.OwnershipName).InsertOne(context.TODO(), ownerItem)
	if err != nil {
		return err
	}
	if result.InsertedID != objectId {
		return fmt.Errorf("Ownership insert for object %v %v inserted different id %v", objectType, objectId, result.InsertedID)
	}
	return nil
}

func makeID() string {
	return utils.RandStringBytesMaskImpr(16)
}

func s3Copy(fs fileaccess.FileAccess, srcBucket string, srcPaths []string, dstBucket string, dstPaths []string, failOnError []bool) {
	var wg sync.WaitGroup

	if len(srcPaths) != len(dstPaths) || len(srcPaths) != len(failOnError) {
		log.Fatalf("s3Copy inputs not the same length for srcBucket %v, dstBucket %v\n", srcBucket, dstBucket)
	}

	// Copy each in its own thread
	for c, srcPath := range srcPaths {
		wg.Add(1)
		go func(srcBucket string, srcPath string, dstBucket string, dstPath string, failOnError bool) {
			defer wg.Done()

			bytes, err := fs.ReadObject(srcBucket, srcPath)
			if err != nil {
				if failOnError {
					log.Fatalln(err)
				} else {
					log.Println(err)
				}
			}

			err = fs.WriteObject(dstBucket, dstPath, bytes)
			if err != nil {
				if failOnError {
					log.Fatalln(err)
				} else {
					log.Println(err)
				}
			}

			fmt.Printf("  Wrote: s3://%v/%v\n", dstBucket, dstPath)
		}(srcBucket, srcPath, dstBucket, dstPaths[c], failOnError[c])
	}

	// Wait for all
	wg.Wait()
}
