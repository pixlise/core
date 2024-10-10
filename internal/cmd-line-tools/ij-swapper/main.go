package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var t0 = time.Now().UnixMilli()

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var destMongoSecret string
	var dbName string

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're modifying i/j's in")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		dbName,
	}
	checkNotEmptyName := []string{
		"dbName",
	}
	for c, s := range checkNotEmpty {
		if len(s) <= 0 {
			log.Fatalf("Parameter: %v was empty", checkNotEmptyName[c])
		}
	}

	// Get a session for the bucket region
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS session. Error: %v", err)
	}

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to mongo
	destMongoClient, _, err := mongoDBConnection.Connect(sess, destMongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	// Destination DB is the new pixlise one
	destDB := destMongoClient.Database(dbName) //mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	// Verify the dataset is valid
	ctx := context.TODO()
	coll := destDB.Collection(dbCollections.ImageBeamLocationsName)

	collWrite := destDB.Collection(dbCollections.ImageBeamLocationsName + "IJSwapped")

	// Find beam locations for each image, if there is one, we swap in all versions stored
	cursor, err := coll.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		log.Fatalln(err)
	}

	beamLocs := []*protos.ImageLocations{}
	err = cursor.All(ctx, &beamLocs)
	if err != nil {
		log.Fatalln(err)
	}

	for _, beamLoc := range beamLocs {
		fmt.Printf("Image: %v\n", beamLoc.ImageName)

		for _, locs := range beamLoc.LocationPerScan {
			fmt.Printf(" Flipping for beam version: %v, instrument: %v\n", locs.BeamVersion, locs.Instrument)

			flippedLoc := []*protos.Coordinate2D{}
			for _, loc := range locs.Locations {
				if loc == nil {
					flippedLoc = append(flippedLoc, nil)
				} else {
					flippedLoc = append(flippedLoc, &protos.Coordinate2D{I: loc.J, J: loc.I})
				}
			}

			// Save in destination table
			locs.Locations = flippedLoc
		}

		writeResult, err := collWrite.InsertOne(ctx, beamLoc)
		if err != nil {
			log.Fatalln(err)
		}

		if writeResult.InsertedID != beamLoc.ImageName {
			fmt.Printf("Write locations: %v saved with unexpected id: %v", beamLoc.ImageName, writeResult.InsertedID)
		}
	}

	printFinishStats()
}

func getWithoutVersion(fileName string) string {
	return fileName[0:len(fileName)-6] + "__" + fileName[len(fileName)-4:]
}

func fatalError(err error) {
	printFinishStats()
	log.Fatal(err)
}

func printFinishStats() {
	t1 := time.Now().UnixMilli()
	sec := (t1 - t0) / 1000
	fmt.Printf("Runtime %v seconds\n", sec)
}
