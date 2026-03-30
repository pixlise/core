package scanOwner

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ReadAutoSharer(creatorUserId string, instrument protos.ScanInstrument, db *mongo.Database, jobLog logger.ILogger) (*protos.ScanAutoShareEntry, error) {
	// Look up who to auto-share with based on creator ID
	coll := db.Collection(dbCollections.ScanAutoShareName)
	optFind := options.FindOne()

	autoShare := &protos.ScanAutoShareEntry{}
	sharer := creatorUserId

	if len(sharer) <= 0 {
		// we dont have a creator, so probably started as an automated process. Here we
		// try to look up the auto-share destination by instrument type
		sharer = instrument.String()
	}

	jobLog.Infof("Looking up auto-share group(s) for: \"%v\"", sharer)

	autoShareResult := coll.FindOne(context.TODO(), bson.D{{Key: "_id", Value: sharer}}, optFind)
	if autoShareResult.Err() != nil {
		// We couldn't find someone to auto-share it with, if we don't have anyone to share with, just fail here
		if autoShareResult.Err() == mongo.ErrNoDocuments {
			// If the user has no auto-share destination configured, share with just the user - BUT if we're
			// not dealing with a user here, we must be importing via the pipeline, in which case it should've
			// been configured to share already...
			if len(creatorUserId) > 0 {
				jobLog.Infof("No auto-share destination found, so only importing user will be able to access this dataset.")
				autoShare.Id = creatorUserId
				autoShare.Viewers = &protos.UserGroupList{UserIds: []string{}, GroupIds: []string{}}
				autoShare.Editors = &protos.UserGroupList{UserIds: []string{creatorUserId}, GroupIds: []string{}}
			} else {
				return nil, fmt.Errorf("Cannot work out groups to auto-share imported dataset with")
			}
		} else {
			return nil, autoShareResult.Err()
		}
	} else {
		err := autoShareResult.Decode(autoShare)

		if err != nil {
			return nil, fmt.Errorf("Failed to decode auto share configuration: %v", err)
		}
	}

	return autoShare, nil
}
