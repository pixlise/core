package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

var t0 = time.Now().UnixMilli()

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var destMongoSecret string
	var dbName string
	var scanId string
	var fileName string
	var sourceDataBucket string

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're importing to")
	flag.StringVar(&scanId, "scanId", "", "Scan ID we're importing for")
	flag.StringVar(&fileName, "fileName", "", "CSV file name to import")
	flag.StringVar(&sourceDataBucket, "sourceDataBucket", "", "Data bucket so we can read dataset.bin file")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		scanId,
		fileName,
		dbName,
	}
	checkNotEmptyName := []string{
		"scanId",
		"fileName",
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

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

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
	coll := destDB.Collection(dbCollections.ScansName)

	result := coll.FindOne(ctx, bson.M{"_id": scanId}, options.FindOne())
	if result.Err() != nil {
		log.Fatalln(result.Err())
	}

	scanItem := protos.ScanItem{}
	if err := result.Decode(&scanItem); err != nil {
		log.Fatalln(result.Err())
	}

	// Get all images for this scan
	coll = destDB.Collection(dbCollections.ImagesName)

	cursor, err := coll.Find(ctx, bson.M{"originscanid": scanId}, options.Find())
	if err != nil {
		log.Fatalln(err)
	}

	images := []*protos.ScanImage{}
	err = cursor.All(ctx, &images)
	if err != nil {
		log.Fatalln(err)
	}

	// Make a PMC->image name lookup
	pmcImageLookup := map[int32]string{}
	for _, img := range images {
		nameBits, err := gdsfilename.ParseFileName(img.ImagePath)
		if err != nil {
			fmt.Printf("Failed to get PMC from image file name: %v. Skipping. Error was: %v\n", img.ImagePath, err)
		} else {
			if nameBits.ProdType != "RCM" {
				fmt.Printf("Skipping image: %v\n", img.ImagePath)
				continue
			}
			pmc, err := nameBits.PMC()
			if err != nil {
				log.Fatalln(err)
			}

			pmcImageLookup[pmc] = img.ImagePath
		}
	}

	// Find out what PMCs we have ij's for, and find the corresponding image file name to import for
	// this way we can import into ImageBeamLocations using the file name, and insert an entry for v3
	beamLocs, err := dataImportHelpers.ReadBeamLocationsFile(fileName, true, 0, []string{"drift_x", "drift_y", "drift_z"}, &logger.StdOutLogger{})

	if err != nil {
		log.Fatalln(err)
	}

	for _, beam := range beamLocs {
		// They should all be the same so only checking first one
		// NOTE: Also ensure we don't have any images stored for PMCs that we don't have beam data for!
		validPMCs := []int32{}
		for imgPMC := range beam.IJ {
			if _, ok := pmcImageLookup[imgPMC]; !ok {
				log.Fatalf("Failed to find image for ij PMC: %v", imgPMC)
			} else {
				validPMCs = append(validPMCs, imgPMC)
			}
		}

		for pmc := range pmcImageLookup {
			if !utils.ItemInSlice(pmc, validPMCs) {
				delete(pmcImageLookup, pmc)
			}
		}
		break
	}

	s3Path := fmt.Sprintf("Scans/%v/dataset.bin", scanId)
	exprBytes, err := fs.ReadObject(sourceDataBucket, s3Path)
	if err != nil {
		log.Fatalln(err)
	}

	exprPB := &protos.Experiment{}
	err = proto.Unmarshal(exprBytes, exprPB)
	if err != nil {
		log.Fatalf("Failed to decode experiment: %v", err)
	}

	// Now construct and save beam location entry. NOTE: it should already exist!
	coll = destDB.Collection(dbCollections.ImageBeamLocationsName)
	for imgPMC, imgName := range pmcImageLookup {
		imgId := bson.D{{Key: "_id", Value: imgName}}
		imgBeamItemResult := coll.FindOne(ctx, imgId, options.FindOne())
		if imgBeamItemResult.Err() != nil {
			log.Fatalln(imgBeamItemResult.Err())
		}

		imageBeamLocations := &protos.ImageLocations{}
		err = imgBeamItemResult.Decode(&imageBeamLocations)
		if err != nil {
			log.Fatalf("Failed to read beam locations for image: %v, scan: %v. Error: %v", imgName, scanItem.Id, err)
		}

		if len(imageBeamLocations.LocationPerScan) != 1 || imageBeamLocations.LocationPerScan[0].ScanId != scanId || imageBeamLocations.LocationPerScan[0].BeamVersion != 2 {
			log.Fatalf("Read beams for image: %v, got unexpected entries, expected one entry for scan %v, v2", imgName, scanId)
		}

		// Now insert this new location set
		ijs := []*protos.Coordinate2D{}

		for _, loc := range exprPB.Locations {
			if loc.Beam == nil {
				ijs = append(ijs, nil)
			} else {
				pmc, err := strconv.Atoi(loc.Id)
				if err != nil {
					log.Fatalf("Failed to decode PMC %v from scan %v: %v", loc.Id, scanId, err)
				}

				if beam, ok := beamLocs[int32(pmc)]; !ok {
					//log.Fatalf("Failed to find v3 beam for PMC: %v\n", pmc)
					fmt.Printf("WARNING: Failed to find v3 beam for PMC: %v, inserting nil\n", pmc)
					ijs = append(ijs, nil)
				} else {
					ijs = append(ijs, &protos.Coordinate2D{I: beam.IJ[imgPMC].I, J: beam.IJ[imgPMC].J})
				}
			}
		}

		loc := &protos.ImageLocationsForScan{
			ScanId:      scanId,
			BeamVersion: 3,
			Instrument:  imageBeamLocations.LocationPerScan[0].Instrument,
			Locations:   ijs,
		}

		updResult, err := coll.UpdateOne(ctx, imgId, bson.D{{Key: "$push", Value: bson.M{"locationperscan": loc}}})
		if err != nil {
			log.Fatalln(err)
		}

		if updResult.MatchedCount != 1 && updResult.ModifiedCount != 1 {
			log.Fatalf("Perhaps beam v3 writing for image: %v didn't work? %+v", imgName, updResult)
		}

		// Confirm
		fmt.Printf("Wrote beam locations for image: %v associated with scan %v, now confirming it worked...\n", imgName, scanId)

		imgBeamItemResult = coll.FindOne(ctx, imgId, options.FindOne())
		if imgBeamItemResult.Err() != nil {
			log.Fatalln(imgBeamItemResult.Err())
		}

		imageBeamLocations = &protos.ImageLocations{}
		err = imgBeamItemResult.Decode(&imageBeamLocations)
		if err != nil {
			log.Fatalf("Failed to read beam locations to confirm writing beams for image: %v, scan: %v. Error: %v", imgName, scanId, err)
		}

		if len(imageBeamLocations.LocationPerScan) != 2 {
			log.Fatalf("Expected 2 stored beam locations for image: %v, scan: %v", imgName, scanId)
		}

		found3 := false
		for _, loc := range imageBeamLocations.LocationPerScan {
			if loc.BeamVersion == 3 {
				found3 = true
				break
			}
		}

		if !found3 {
			log.Fatalf("Expected to find stored beam location v3 for image: %v, scan: %v", imgName, scanId)
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
