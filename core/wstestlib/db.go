package wstestlib

import (
	"github.com/pixlise/core/v4/core/logger"
	mongoDBConnection "github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
	"go.mongodb.org/mongo-driver/mongo"
)

var db *mongo.Database

var dbSuffix = ""

func GetDB() *mongo.Database {
	// If we don't have one yet, generate one, but don't make these infinitely keep going... This is purely to reduce clashes when
	// running tests in builds - multiple branches running concurrently causes tests to have race conditions adding/deleting items
	// and here we try to reduce the chances of this occuring
	if len(dbSuffix) <= 0 {
		dbSuffix = utils.RandStringBytesMaskImpr(1)
	}

	return GetDBWithSuffix(dbSuffix)
}

func GetDBWithSuffix(suffix string) *mongo.Database {
	if db == nil {
		// Connect to a local one ONLY
		logger := logger.StdOutLogger{}
		logger.SetLogLevel(2)

		client, err := mongoDBConnection.Connect(nil, "", &logger)
		if err != nil {
			// This meant it was hard to catch when it was failing because DB not running locally
			//log.Fatal(err)
			panic(err)
		}

		if len(suffix) > 0 {
			suffix = "_" + suffix
		}

		dbName := mongoDBConnection.GetDatabaseName("pixlise", "unittest"+suffix)
		db = client.Database(dbName)
	}

	return db
}
