package scanOwner

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func WriteAutoSharedOwnership(
	id string,
	objType protos.ObjectType,
	autoShare *protos.ScanAutoShareEntry,
	creatorUserId string,
	creationUnixTimeSec int64,
	db *mongo.Database,
	jobLog logger.ILogger) error {

	ownerItem := wsHelpers.MakeOwnerForWrite(id, objType, creatorUserId, creationUnixTimeSec)

	ownerItem.Viewers = autoShare.Viewers
	ownerItem.Editors = autoShare.Editors

	coll := db.Collection(dbCollections.OwnershipName)
	opt := options.Update().SetUpsert(true)

	jobLog.Infof("Writing ownership to DB for scan %v...", ownerItem.Id)
	_, err := coll.UpdateOne(context.TODO(), bson.D{{Key: "_id", Value: ownerItem.Id}}, bson.D{{Key: "$set", Value: ownerItem}}, opt)
	if err != nil {
		jobLog.Errorf("Failed to write ownership item to DB: %v", err)
		return err
	}

	return nil
}
