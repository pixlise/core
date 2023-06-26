package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/mongoDBConnection"
)

var maxItemsToRead int

func main() {
	var sourceMongoSecret string
	var destMongoSecret string
	var dataBucket string
	var configBucket string
	var userContentBucket string
	var srcEnvName string
	var destEnvName string

	flag.StringVar(&sourceMongoSecret, "sourceMongoSecret", "", "Source mongo DB secret")
	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dataBucket, "dataBucket", "", "Data bucket")
	flag.StringVar(&userContentBucket, "userContentBucket", "", "User content bucket")
	flag.StringVar(&configBucket, "configBucket", "", "Config bucket")
	flag.StringVar(&srcEnvName, "srcEnvName", "", "Source Environment Name")
	flag.StringVar(&destEnvName, "destEnvName", "", "Destination Environment Name")
	flag.IntVar(&maxItemsToRead, "maxItems", 0, "Max number of items to read into any table, 0=unlimited")

	flag.Parse()

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
	sourceMongoClient, err := mongoDBConnection.Connect(sess, sourceMongoSecret, iLog)
	if err != nil {
		log.Fatal(err)
	}

	destMongoClient, err := mongoDBConnection.Connect(sess, destMongoSecret, iLog)
	if err != nil {
		log.Fatal(err)
	}

	// Destination DB is the new pixlise one
	srcExprDB := sourceMongoClient.Database(mongoDBConnection.GetDatabaseName("expressions", srcEnvName))
	srcUserDB := sourceMongoClient.Database(mongoDBConnection.GetDatabaseName("userdatabase", srcEnvName))

	// Destination DB is the new pixlise one
	destDB := destMongoClient.Database(mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	fmt.Println("==========================================")
	fmt.Println("Migrating data from old users DB...")
	fmt.Println("==========================================")
	err = migrateUsersDB(srcUserDB, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("==========================================")
	fmt.Println("Migrating data from old expressions DB...")
	fmt.Println("==========================================")
	err = migrateExpressionsDB(srcExprDB, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("==========================================")
	fmt.Println("Migrating data from config bucket...")
	fmt.Println("==========================================")
	fmt.Println("Detector Configs...")
	err = migrateDetectorConfigs(configBucket, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Piquant Version...")
	err = migratePiquantVersion(configBucket, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	/* Decided to leave PIQUANT configs in S3 because that way PIQUANT docker container has authenticated direct access
	fmt.Println("Piquant Configs...")
	err = migratePiquantConfigs(configBucket, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}*/

	// List all of S3 user contents
	fmt.Println("Listing user contents from S3...")
	userContentPaths, err := fs.ListObjects(userContentBucket, filepaths.RootUserContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("  Listed %v files\n", len(userContentPaths))

	fmt.Println("==========================================")
	fmt.Println("Migrating data from user content bucket...")
	fmt.Println("==========================================")

	fmt.Println("Element Sets...")
	err = migrateElementSets(userContentBucket, userContentPaths, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("ROIs...")
	err = migrateROIs(userContentBucket, userContentPaths, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("RGB Mixes...")
	err = migrateRGBMixes(userContentBucket, userContentPaths, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Tags...")
	err = migrateTags(userContentBucket, userContentPaths, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Diffraction Peak...")
	err = migrateDiffraction(userContentBucket, userContentPaths, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}
		fmt.Println("View States...")
		err = migrateViewStates(userContentBucket, userContentPaths, fs, destDB)
		if err != nil {
			log.Fatal(err)
	}

	fmt.Println("==========================================")
	fmt.Println("Migrating data from datasets bucket...")
	fmt.Println("==========================================")

	fmt.Println("Datasets...")
	err = migrateDatasets(configBucket, dataBucket, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}

	// Quants
}
