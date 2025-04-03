package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var t0 = time.Now().UnixMilli()

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var mongoSecret string
	var dbName string

	flag.StringVar(&mongoSecret, "mongoSecret", "", "Mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're importing to")

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
	iLog := &logger.StdErrLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to mongo
	mongoClient, _, err := mongoDBConnection.Connect(sess, mongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	// Destination DB is the new pixlise one
	destDB := mongoClient.Database(dbName) //mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	// Verify the dataset is valid
	ctx := context.TODO()
	coll := destDB.Collection(dbCollections.ImageBeamLocationsName)
	opt := options.Find().SetProjection(bson.D{
		{Key: "id", Value: true},
	})

	result, err := coll.Find(ctx, bson.M{}, opt)
	if err != nil {
		fatalError(err)
	}

	imageBeams := []*protos.ImageLocations{}
	if err := result.All(ctx, &imageBeams); err != nil {
		fatalError(result.Err())
	}

	// We want to read the images in alphabetical order to ensure we read image versions in order
	imageNames := []string{}
	for _, img := range imageBeams {
		imageNames = append(imageNames, img.ImageName)
	}

	sort.Strings(imageNames)

	newColl := destDB.Collection(dbCollections.ImageBeamLocationsName + "VersionFree")
	err = newColl.Drop(ctx)
	if err != nil {
		fatalError(err)
	}

	// For each ID, if it's NOT a PDS encoded file name, store it
	// If it is PDS encoded file name we check if we have a version-less one stored already
	for _, origImageName := range imageNames {
		imgReadResult := coll.FindOne(ctx, bson.D{{Key: "_id", Value: origImageName}}, options.FindOne())
		if imgReadResult.Err() != nil {
			fatalError(imgReadResult.Err())
		}

		img := &protos.ImageLocations{}
		if err := imgReadResult.Decode(img); err != nil {
			log.Fatalln(err)
		}

		meta, err := gdsfilename.ParseFileName(img.ImageName)
		if err == nil {
			// Snip off the version and see if this is stored already
			meta.SetVersionStr("__")

			if len(meta.FilePath) <= 0 {
				log.Fatalf("Expected path in image name: %v\n", img.ImageName)
			}

			img.ImageName = meta.ToString(true, true)

			// If there is an existing one, we have to update it, otherwise just write this as the record
			existingImgRes := newColl.FindOne(ctx, bson.D{{Key: "_id", Value: img.ImageName}})
			if existingImgRes.Err() != nil {
				if existingImgRes.Err() != mongo.ErrNoDocuments {
					log.Fatalf("%v: %v", origImageName, existingImgRes.Err())
				}

				// Fall through: Let it be written as is
			} else {
				// There's an existing one, update that one with any new beams we may have stored
				existingImg := &protos.ImageLocations{}
				if err = existingImgRes.Decode(existingImg); err != nil {
					log.Fatalf("%v: %v", origImageName, err)
				}

				for _, loc := range img.LocationPerScan {
					found := false
					//foundEquals := true
					for locIdx, existingLoc := range existingImg.LocationPerScan {
						if loc.BeamVersion == existingLoc.BeamVersion && loc.ScanId == existingLoc.ScanId {
							found = true

							// Already exists, check if they're equal
							for c, l := range loc.Locations {
								el := existingLoc.Locations[c]
								replaceExisting := false

								if el == nil && l == nil {
									continue
								} else if el != nil && l != nil {
									if el.I != l.I || el.J != l.J {
										log.Printf("WARNING: Image %v beam version %v doesn't match image %v, beam version %v at idx: %v\n", origImageName, loc.BeamVersion, existingImg.ImageName, existingLoc.BeamVersion, c)
										replaceExisting = true
									}
								} else {
									log.Printf("WARNING: Image %v beam version %v doesn't match image %v, beam version %v at idx: %v, one is nil, one is not\n", origImageName, loc.BeamVersion, existingImg.ImageName, existingLoc.BeamVersion, c)
									replaceExisting = true
								}

								if replaceExisting {
									// At this point, delete what's already there, we will replace it with the newer image versions beam locations
									existingImg.LocationPerScan = append(existingImg.LocationPerScan[0:locIdx], existingImg.LocationPerScan[locIdx+1:]...)

									found = false // We removed it, so say it's not found (allowing it to be added now)
									break
								}
							}
							break
						}
					}

					if !found {
						// Add it
						existingImg.LocationPerScan = append(existingImg.LocationPerScan, loc)
					}
				}

				img = existingImg
			}
		}
		// else: Must be a non-PDS file name, just store it in the new collection

		insResult, err := newColl.UpdateByID(ctx, img.ImageName, bson.D{{Key: "$set", Value: img}}, options.Update().SetUpsert(true))
		if err != nil {
			fatalError(err)
		} else if insResult.UpsertedCount <= 0 && insResult.MatchedCount <= 0 && insResult.ModifiedCount <= 0 {
			log.Printf("Unexpected result for beam location upsert: %v. %+v\n", img.ImageName, insResult)
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
