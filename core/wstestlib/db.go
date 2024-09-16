package wstestlib

import (
	"github.com/pixlise/core/v4/core/logger"
	mongoDBConnection "github.com/pixlise/core/v4/core/mongoDBConnection"
	"go.mongodb.org/mongo-driver/mongo"
)

var db *mongo.Database

func GetDB() *mongo.Database {
	return GetDBWithEnvironment("unittest")
}

func GetDBWithEnvironment(envName string) *mongo.Database {
	if db == nil {
		// Connect to a local one ONLY
		logger := logger.StdOutLogger{}
		logger.SetLogLevel(2)

		client, _, err := mongoDBConnection.Connect(nil, "", &logger)
		if err != nil {
			// This meant it was hard to catch when it was failing because DB not running locally
			//log.Fatal(err)
			panic(err)
		}

		dbName := mongoDBConnection.GetDatabaseName("pixlise", envName)
		db = client.Database(dbName)
	}

	return db
}
