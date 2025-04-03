package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/beamLocation"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

var t0 = time.Now().UnixMilli()

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var destMongoSecret string
	var sourceDataBucket string
	var destEnvName string

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&sourceDataBucket, "sourceDataBucket", "", "Data bucket")
	flag.StringVar(&destEnvName, "destEnvName", "", "Destination Environment Name")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		sourceDataBucket,
		destEnvName,
	}
	checkNotEmptyName := []string{
		"sourceDataBucket",
		"destEnvName",
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

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	iLog := &logger.StdErrLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to mongo
	destMongoClient, _, err := mongoDBConnection.Connect(sess, destMongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	// Destination DB is the new pixlise one
	destDB := destMongoClient.Database("prodCopy") //mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	// Read the dataset ids that we want to find v1 geometry for
	ctx := context.TODO()
	coll := destDB.Collection(dbCollections.ScansName)

	cursor, err := coll.Find(ctx, bson.D{}, options.Find())
	if err != nil {
		log.Fatalln(err)
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(ctx, &scans)
	if err != nil {
		return
	}

	for _, scanItem := range scans {
		if scanItem.Instrument == protos.ScanInstrument_PIXL_FM && scanItem.ContentCounts["NormalSpectra"] > 0 {
			log.Printf("Reading: %v [Sol %v - %v]", scanItem.Id, scanItem.Meta["Sol"], scanItem.Title)

			if sol, ok := scanItem.Meta["Sol"]; !ok || len(sol) <= 0 {
				log.Printf("  SKIPPING scan %v: doesn't contain a sol in its meta data", scanItem.Id)
				continue
			} else {
				if sol[0] < '0' || sol[0] > '9' {
					log.Printf("  SKIPPING scan %v: sol %v is not valid", scanItem.Id, scanItem.Meta["Sol"])
					continue
				}
			}

			s3Path := fmt.Sprintf("Datasets/%v/dataset.bin", scanItem.Id)
			exprBytes, err := fs.ReadObject(sourceDataBucket, s3Path)
			if err != nil {
				if fs.IsNotFoundError((err)) {
					log.Printf("  SKIPPING scan %v: no data for this with v1 coordinates, maybe it's newer", scanItem.Id)
					continue
				}

				log.Fatalln(err)
			}

			exprPB := &protos.Experiment{}
			err = proto.Unmarshal(exprBytes, exprPB)
			if err != nil {
				log.Fatalf("Failed to decode experiment: %v", err)
			}

			// Also read all images we have for this scan id
			beamColl := destDB.Collection(dbCollections.ImageBeamLocationsName)
			beamFilter := bson.D{{Key: "locationperscan.0.scanid", Value: scanItem.Id}}

			imgBeamItemResult, err := beamColl.Find(ctx, beamFilter, options.Find())

			if err != nil {
				if err == mongo.ErrNoDocuments {
					log.Printf("  SKIPPING scan %v: No image beam locations found", scanItem.Id)
					continue
				} else {
					log.Fatalf("Error when reading scan %v beam locations: %v", scanItem.Id, err)
				}
			}

			imageBeamLocations := []*protos.ImageLocations{}
			err = imgBeamItemResult.All(ctx, &imageBeamLocations)
			if err != nil {
				log.Fatalf("Failed to read beam locations for scan: %v. Error: %v", scanItem.Id, err)
			}

			for alignedIdx, img := range exprPB.AlignedContextImages {
				imgId := fmt.Sprintf("%v/%v", scanItem.Id, img.Image)

				// Make sure we have an entry for this already. NOTE: we're comparing by ignoring the version number!
				imgIdSansVersion := getWithoutVersion(imgId)

				var matchedBeamLocation *protos.ImageLocations
				for _, loc := range imageBeamLocations {
					thisLocWithoutVersion := getWithoutVersion(loc.ImageName)
					if thisLocWithoutVersion == imgIdSansVersion {
						matchedBeamLocation = loc
					}
				}

				if matchedBeamLocation == nil {
					log.Printf("  SKIPPING %v: No beam locations found", imgId)
					continue
				}

				// Make sure there are no v1's already stored
				if len(matchedBeamLocation.LocationPerScan) != 1 {
					log.Printf("  SKIPPING %v: Beam Location had wrong count: %v", imgId, len(matchedBeamLocation.LocationPerScan))
					continue
				}

				if matchedBeamLocation.LocationPerScan[0].BeamVersion != 2 {
					log.Fatalf("Beam Location for %v did not contain expected version 2", imgId)
				}

				// Double check some more stuff
				if matchedBeamLocation.LocationPerScan[0].Instrument != scanItem.Instrument {
					log.Fatalf("Beam Location for %v did not contain expected instrument", imgId)
				}
				if matchedBeamLocation.LocationPerScan[0].ScanId != scanItem.Id {
					log.Fatalf("Beam Location for %v did not contain expected scanId", imgId)
				}

				ijs := beamLocation.ReadIJs(alignedIdx, exprPB)

				// Check that they differ
				if len(ijs) != len(matchedBeamLocation.LocationPerScan[0].Locations) {
					log.Fatalf("Beam count from DB (%v) doesn't match beam count from experiment file (%v) for image: %v", len(matchedBeamLocation.LocationPerScan[0].Locations), len(ijs), imgId)
				}

				equalCount := 0
				for c := 0; c < len(ijs); c++ {
					if (ijs[c] == nil && matchedBeamLocation.LocationPerScan[0].Locations[c] == nil) ||
						(ijs[c].I == matchedBeamLocation.LocationPerScan[0].Locations[c].I && ijs[c].J == matchedBeamLocation.LocationPerScan[0].Locations[c].J) {
						equalCount++
					}
				}

				if equalCount > len(ijs)/2 {
					log.Printf("  SKIPPING %v: Beam v2 is too similar to v1", imgId)
					continue
				}

				// Set up an update for this so we just add to the existing array of beam locations in the DB
				matchedBeamLocation.LocationPerScan = append(matchedBeamLocation.LocationPerScan, &protos.ImageLocationsForScan{
					ScanId:      scanItem.Id,
					BeamVersion: 1,
					Instrument:  scanItem.Instrument,
					Locations:   ijs,
				})

				// Write it back
				beamFilter = bson.D{{Key: "_id", Value: matchedBeamLocation.ImageName}}
				updResult, err := beamColl.ReplaceOne(ctx, beamFilter, matchedBeamLocation, options.Replace())

				if err != nil {
					log.Fatalf("Failed to import beam for: %v. Error: %v", imgId, err)
				}

				if updResult.ModifiedCount != 1 || updResult.MatchedCount != 1 {
					log.Fatalf("Got unexpected replace result for: %v. %+v", imgId, updResult)
				}

				log.Printf("SUCCESS importing v1 for: %v", imgId)
			}
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
