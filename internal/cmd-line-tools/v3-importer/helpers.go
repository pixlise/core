package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func fixUserId(userId string) string {
	if !strings.HasPrefix(userId, "auth0|") {
		return "auth0|" + userId
	}
	return userId
}

func saveOwnershipItem(objectId string, objectType protos.ObjectType, userId string, timeStampUnixSec uint64, dest *mongo.Database) error {
	userId = fixUserId(userId)

	ownerItem := &protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatorUserId:  userId,
		CreatedUnixSec: timeStampUnixSec,
		//Viewers: ,
		Editors: &protos.UserGroupList{
			UserIds: []string{userId},
		},
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
