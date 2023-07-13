package wstestlib

import (
	"log"

	"github.com/pixlise/core/v3/core/logger"
	mongoDBConnection "github.com/pixlise/core/v3/core/mongoDBConnection"
	"go.mongodb.org/mongo-driver/mongo"
)

var db *mongo.Database

func GetDB() *mongo.Database {
	if db == nil {
		// Connect to a local one ONLY
		logger := logger.StdOutLogger{}
		logger.SetLogLevel(2)

		client, err := mongoDBConnection.Connect(nil, "", &logger)
		if err != nil {
			log.Fatal(err)
		}

		dbName := mongoDBConnection.GetDatabaseName("pixlise", "unittest")
		db = client.Database(dbName)
	}

	return db
}
