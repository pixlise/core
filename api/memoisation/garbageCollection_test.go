package memoisation

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Example_memoisation_CollectGarbage() {
	db := wstestlib.GetDB()
	ctx := context.TODO()
	coll := db.Collection(dbCollections.MemoisedItemsName)

	// Insert an item that's too old, and one that's newly accessed
	ts := &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890, 1234567890, 1234567890},
	}

	now := uint32(ts.GetTimeNowSec())
	maxAge := uint32(100)

	item := &protos.MemoisedItem{
		Key:                 "key123",
		MemoTimeUnixSec:     now - maxAge - 50,
		Data:                []byte{1, 3, 5, 7},
		ScanId:              "scan333",
		DataSize:            uint32(4),
		LastReadTimeUnixSec: now - maxAge - 10,
	}

	opt := options.Update().SetUpsert(true)
	_, err := coll.UpdateByID(ctx, item.Key, bson.D{{Key: "$set", Value: item}}, opt)
	fmt.Printf("Insert 1: %v\n", err)

	item = &protos.MemoisedItem{
		Key:                 "key456",
		MemoTimeUnixSec:     now - maxAge - 55,
		Data:                []byte{2, 4, 6, 8, 10},
		ScanId:              "scan222",
		DataSize:            uint32(5),
		LastReadTimeUnixSec: now - 5,
	}
	_, err = coll.UpdateByID(ctx, item.Key, bson.D{{Key: "$set", Value: item}}, opt)
	fmt.Printf("Insert 2: %v\n", err)

	item = &protos.MemoisedItem{
		Key:                 "key789",
		MemoTimeUnixSec:     now - maxAge - 60,
		Data:                []byte{10, 20, 30},
		ScanId:              "scan222",
		DataSize:            uint32(5),
		LastReadTimeUnixSec: now - 5,
		NoGC:                true,
	}
	_, err = coll.UpdateByID(ctx, item.Key, bson.D{{Key: "$set", Value: item}}, opt)
	fmt.Printf("Insert 3: %v\n", err)

	log := &logger.StdOutLogger{}
	// Should delete one based on time
	collectGarbage(db, maxAge, ts, log)

	// Should delete the other based on time, no GC should stay
	maxAge = 1
	collectGarbage(db, maxAge, ts, log)

	// Output:
	// Insert 1: <nil>
	// Insert 2: <nil>
	// Insert 3: <nil>
	// INFO: Memoisation GC starting...
	// INFO: Memoisation GC deleted 1 items
	// INFO: Memoisation GC starting...
	// INFO: Memoisation GC deleted 1 items
}
