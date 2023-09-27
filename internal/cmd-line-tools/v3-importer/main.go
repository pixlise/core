package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/mongoDBConnection"
)

var maxItemsToRead int
var quantLogLimitCount int

func main() {
	rand.Seed(time.Now().UnixNano())

	t0 := time.Now().UnixMilli()
	fmt.Printf("Started: %v\n", time.Now().String())

	var sourceMongoSecret string
	var destMongoSecret string
	var dataBucket string
	var destDataBucket string
	var configBucket string
	var userContentBucket string
	var destUserContentBucket string
	var srcEnvName string
	var destEnvName string
	var auth0Domain, auth0ClientId, auth0Secret string
	var limitToDatasetIDs string

	flag.StringVar(&sourceMongoSecret, "sourceMongoSecret", "", "Source mongo DB secret")
	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dataBucket, "dataBucket", "", "Data bucket")
	flag.StringVar(&destDataBucket, "destDataBucket", "", "Destination data bucket")
	flag.StringVar(&userContentBucket, "userContentBucket", "", "User content bucket")
	flag.StringVar(&destUserContentBucket, "destUserContentBucket", "", "Destination user content bucket")
	flag.StringVar(&configBucket, "configBucket", "", "Config bucket")
	flag.StringVar(&srcEnvName, "srcEnvName", "", "Source Environment Name")
	flag.StringVar(&destEnvName, "destEnvName", "", "Destination Environment Name")
	flag.IntVar(&maxItemsToRead, "maxItems", 0, "Max number of items to read into any table, 0=unlimited")
	flag.StringVar(&auth0Domain, "auth0Domain", "", "Auth0 domain for management API")
	flag.StringVar(&auth0ClientId, "auth0ClientId", "", "Auth0 client id for management API")
	flag.StringVar(&auth0Secret, "auth0Secret", "", "Auth0 secret for management API")
	flag.StringVar(&limitToDatasetIDs, "limitToDatasetIDs", "", "Comma-separated dataset IDs to limit import to (for speed/testing)")
	flag.IntVar(&quantLogLimitCount, "quantLogLimitCount", 0, "Limits how many log files are copied (for speed/testing)")

	flag.Parse()

	limitToDatasetIDsList := strings.Split(limitToDatasetIDs, ",")

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

	// Clear out ownership table first
	err = destDB.Collection(dbCollections.OwnershipName).Drop(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	var listingWG sync.WaitGroup
	listingWG.Add(2)

	userGroups := map[string]string{}
	go func() {
		defer listingWG.Done()
		fmt.Println("==========================================")
		fmt.Println("Migrating data from old users DB...")
		fmt.Println("==========================================")
		err := migrateUsersDB(srcUserDB, destDB)
		if err != nil {
			log.Fatal(err)
		}

		if len(auth0Domain) > 0 && len(auth0ClientId) > 0 && len(auth0Secret) > 0 {
			fmt.Println("==========================================")
			fmt.Println("Migrating user groups from Auth0...")
			fmt.Println("==========================================")
			userGroups, err = migrateAuth0UserGroups(auth0Domain, auth0ClientId, auth0Secret, destDB)
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Println("==========================================")
		fmt.Println("Migrating data from datasets bucket...")
		fmt.Println("==========================================")

		fmt.Println("Datasets...")
		err = migrateDatasets(configBucket, dataBucket, destDataBucket, fs, destDB, limitToDatasetIDsList, userGroups)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("==========================================")
		fmt.Println("Migrating data from old expressions DB...")
		fmt.Println("==========================================")
		err = migrateExpressionsDB(srcExprDB, destDB, userGroups)
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
	}()

	/* Decided to leave PIQUANT configs in S3 because that way PIQUANT docker container has authenticated direct access
	fmt.Println("Piquant Configs...")
	err = migratePiquantConfigs(configBucket, fs, destDB)
	if err != nil {
		log.Fatal(err)
	}*/

	userContentPaths := []string{}
	go func() {
		defer listingWG.Done()
		// List all of S3 user contents
		fmt.Println("Listing user contents from S3...")
		var err error
		userContentPaths, err = fs.ListObjects(userContentBucket, filepaths.RootUserContent)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  Listed %v files\n", len(userContentPaths))
	}()

	listingWG.Wait()

	fmt.Println("==========================================")
	fmt.Println("Migrating data from user content bucket...")
	fmt.Println("==========================================")

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Quant Z-stacks...")
		err = migrateMultiQuants(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Quants...")
		err = migrateQuants(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, destUserContentBucket, userGroups)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Element Sets...")
		err = migrateElementSets(userContentBucket, userContentPaths, fs, destDB, userGroups)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("ROIs...")
		err = migrateROIs(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, userGroups)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("RGB Mixes...")
		err = migrateRGBMixes(userContentBucket, userContentPaths, fs, destDB, userGroups)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Tags...")
		err = migrateTags(userContentBucket, userContentPaths, fs, destDB)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("Diffraction Peak...")
		err = migrateDiffraction(userContentBucket, userContentPaths, fs, destDB)
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Println("View States...")
		err = migrateViewStates(userContentBucket, userContentPaths, fs, destDB)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Wait for all
	wg.Wait()

	t1 := time.Now().UnixMilli()
	sec := (t1 - t0) / 1000

	fmt.Printf("Finished: %v\n", time.Now().String())
	fmt.Printf("Runtime %v seconds\n", sec)
}
