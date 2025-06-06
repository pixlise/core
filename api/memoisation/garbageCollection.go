package memoisation

import (
	"context"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func RunMemoisationGarbageCollector(intervalSec uint32, oldestAllowedSec uint32, mongoDB *mongo.Database, ts timestamper.ITimeStamper, log logger.ILogger) {
	for range time.Tick(time.Second * time.Duration(intervalSec)) {
		collectGarbage(mongoDB, oldestAllowedSec, ts, log)
	}
}

func collectGarbage(mongoDB *mongo.Database, oldestAllowedSec uint32, ts timestamper.ITimeStamper, log logger.ILogger) {
	log.Infof("Memoisation GC starting...")

	oldestAllowedUnixSec := ts.GetTimeNowSec() - int64(oldestAllowedSec)

	ctx := context.TODO()
	opts := options.Delete()
	filter := bson.M{"lastreadtimeunixsec": bson.M{"$lt": oldestAllowedUnixSec}, "nogc": false}
	coll := mongoDB.Collection(dbCollections.MemoisedItemsName)

	delResult, err := coll.DeleteMany(ctx, filter, opts)
	if err != nil {
		log.Errorf("Memoisation GC delete error: %v", err)
	} else {
		log.Infof("Memoisation GC deleted %v items", delResult.DeletedCount)
	}
}
