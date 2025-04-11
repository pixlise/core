package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var maxItemsToRead int
var quantLogLimitCount int

var t0 = time.Now().UnixMilli()

func main() {
	defer reportFailedTasks()
	//rand.Seed(time.Now().UnixNano())

	fmt.Printf("Started: %v\n", time.Now().String())

	var sourceMongoSecret string
	var destMongoSecret string
	var dataBucket string
	var destDataBucket string
	var configBucket string
	var destConfigBucket string
	var userContentBucket string
	var destUserContentBucket string
	var srcEnvName string
	var destEnvName string
	var auth0Domain, auth0ClientId, auth0Secret string
	var limitToDatasetIDs string
	var migrateDatasetsEnabled bool
	var migrateROIsEnabled bool
	var migrateDiffractionPeaksEnabled bool
	var migrateRGBMixesEnabled bool
	var migrateTagsEnabled bool
	var migrateQuantsEnabled bool
	var migrateElementSetsEnabled bool
	var migrateZStacksEnabled bool
	var migrateExpressionsEnabled bool
	var fixROISharingMode bool
	var fixROIIndexesMode bool
	var jsonImportDir string
	var migrateOrphanedSharedROIsFile string

	flag.StringVar(&sourceMongoSecret, "sourceMongoSecret", "", "Source mongo DB secret")
	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dataBucket, "dataBucket", "", "Data bucket")
	flag.StringVar(&destDataBucket, "destDataBucket", "", "Destination data bucket")
	flag.StringVar(&userContentBucket, "userContentBucket", "", "User content bucket")
	flag.StringVar(&destUserContentBucket, "destUserContentBucket", "", "Destination user content bucket")
	flag.StringVar(&configBucket, "configBucket", "", "Config bucket")
	flag.StringVar(&destConfigBucket, "destConfigBucket", "", "Destination config bucket (some files are copied)")
	flag.StringVar(&srcEnvName, "srcEnvName", "", "Source Environment Name")
	flag.StringVar(&destEnvName, "destEnvName", "", "Destination Environment Name")
	flag.IntVar(&maxItemsToRead, "maxItems", 0, "Max number of items to read into any table, 0=unlimited")
	flag.StringVar(&auth0Domain, "auth0Domain", "", "Auth0 domain for management API")
	flag.StringVar(&auth0ClientId, "auth0ClientId", "", "Auth0 client id for management API")
	flag.StringVar(&auth0Secret, "auth0Secret", "", "Auth0 secret for management API")
	flag.StringVar(&limitToDatasetIDs, "limitToDatasetIDs", "", "Comma-separated dataset IDs to limit import to (for speed/testing)")
	flag.IntVar(&quantLogLimitCount, "quantLogLimitCount", 0, "Limits how many log files are copied (for speed/testing)")
	flag.BoolVar(&migrateDatasetsEnabled, "migrateDatasetsEnabled", true, "Should we migrate datasets?")
	flag.BoolVar(&migrateROIsEnabled, "migrateROIsEnabled", true, "Should we migrate ROIs?")
	flag.BoolVar(&migrateDiffractionPeaksEnabled, "migrateDiffractionPeaksEnabled", true, "Should we migrate Diffraction Peaks?")
	flag.BoolVar(&migrateRGBMixesEnabled, "migrateRGBMixesEnabled", true, "Should we migrate RGB Mixes?")
	flag.BoolVar(&migrateTagsEnabled, "migrateTagsEnabled", true, "Should we migrate Tags?")
	flag.BoolVar(&migrateQuantsEnabled, "migrateQuantsEnabled", true, "Should we migrate Quants?")
	flag.BoolVar(&migrateElementSetsEnabled, "migrateElementSetsEnabled", true, "Should we migrate Element Sets?")
	flag.BoolVar(&migrateZStacksEnabled, "migrateZStacksEnabled", true, "Should we migrate Z-Stacks?")
	flag.BoolVar(&migrateExpressionsEnabled, "migrateExpressionsEnabled", true, "Should we migrate expressions?")
	flag.BoolVar(&fixROISharingMode, "fixROISharingMode", false, "Fixing ROI sharing states (this is a whole separate mode, won't migrate other things)")
	flag.BoolVar(&fixROIIndexesMode, "fixROIIndexesMode", false, "Fixing ROI indexes (this is a whole separate mode, won't migrate other things)")
	flag.StringVar(&migrateOrphanedSharedROIsFile, "migrateOrphanedSharedROIsFile", "", "Migrates shared ROIs whose id exists in the file (this is a whole separate mode, won't migrate other things)")
	flag.StringVar(&jsonImportDir, "jsonImportDir", "", "If not empty, this is expected to be a directory to read JSON data into DB from. File names must be collection name.")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		dataBucket,
		destDataBucket,
		configBucket,
		destConfigBucket,
		userContentBucket,
		destUserContentBucket,
		srcEnvName,
		destEnvName,
		auth0Domain,
		auth0ClientId,
		auth0Secret,
	}
	checkNotEmptyName := []string{
		"dataBucket",
		"destDataBucket",
		"configBucket",
		"destConfigBucket",
		"userContentBucket",
		"destUserContentBucket",
		"srcEnvName",
		"destEnvName",
		"auth0Domain",
		"auth0ClientId",
		"auth0Secret",
	}
	for c, s := range checkNotEmpty {
		if len(s) <= 0 {
			log.Fatalf("Parameter: %v was empty", checkNotEmptyName[c])
		}
	}

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
	iLog := &logger.StdErrLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to mongo
	sourceMongoClient, _, err := mongoDBConnection.Connect(sess, sourceMongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	destMongoClient, _, err := mongoDBConnection.Connect(sess, destMongoSecret, iLog)
	if err != nil {
		fatalError(err)
	}

	// Destination DB is the new pixlise one
	srcExprDB := sourceMongoClient.Database(mongoDBConnection.GetDatabaseName("expressions", srcEnvName))
	srcUserDB := sourceMongoClient.Database(mongoDBConnection.GetDatabaseName("userdatabase", srcEnvName))

	// Destination DB is the new pixlise one
	destDB := destMongoClient.Database(mongoDBConnection.GetDatabaseName("pixlise", destEnvName))

	if fixROISharingMode {
		fixROISharing(auth0Domain, auth0ClientId, auth0Secret, fs, userContentBucket, limitToDatasetIDsList, destDB)
		return
	}

	if fixROIIndexesMode {
		fixROIIndexes(dataBucket, fs, destDB)
		return
	}

	if len(migrateOrphanedSharedROIsFile) > 0 {
		runMigrateOrphanedSharedROIs(migrateOrphanedSharedROIsFile, fs, userContentBucket, limitToDatasetIDsList, destDB)
		return
	}

	// Clear out ownership table first, but only for the bits we're about to import
	if migrateDatasetsEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_SCAN)
	}
	if migrateROIsEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_ROI)
	}
	if migrateRGBMixesEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_EXPRESSION_GROUP)
	}
	if migrateQuantsEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_QUANTIFICATION)
	}
	if migrateElementSetsEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_ELEMENT_SET)
	}
	if migrateExpressionsEnabled {
		clearOwnership(destDB, protos.ObjectType_OT_EXPRESSION)
		clearOwnership(destDB, protos.ObjectType_OT_DATA_MODULE)
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
			fatalError(err)
		}

		if len(auth0Domain) > 0 && len(auth0ClientId) > 0 && len(auth0Secret) > 0 {
			fmt.Println("==========================================")
			fmt.Println("Migrating user groups from Auth0...")
			fmt.Println("==========================================")
			userGroups, err = migrateAuth0UserGroups(auth0Domain, auth0ClientId, auth0Secret, destDB)
			if err != nil {
				fatalError(err)
			}
		}

		if migrateExpressionsEnabled {
			fmt.Println("==========================================")
			fmt.Println("Migrating data from old expressions DB...")
			fmt.Println("==========================================")
			err = migrateExpressionsDB(srcExprDB, destDB, userGroups)
			if err != nil {
				fatalError(err)
			}
		} else {
			fmt.Println("Skipping migration of expressions...")
		}

		if migrateDatasetsEnabled {
			fmt.Println("==========================================")
			fmt.Println("Migrating data from datasets bucket...")
			fmt.Println("==========================================")

			fmt.Println("Datasets...")
			err = migrateDatasets(configBucket, dataBucket, destDataBucket, fs, destDB, limitToDatasetIDsList, userGroups)
			if err != nil {
				fatalError(err)
			}
		} else {
			fmt.Println("Skipping migration of datasets...")
		}

		fmt.Println("==========================================")
		fmt.Printf("Importing JSON files from %v\n", jsonImportDir)
		fmt.Println("==========================================")
		err = importJSONFiles(jsonImportDir, destDB)
		if err != nil {
			fatalError(err)
		}

		fmt.Println("==========================================")
		fmt.Println("Migrating data from config bucket...")
		fmt.Println("==========================================")
		fmt.Println("Detector Configs...")
		err = migrateDetectorConfigs(configBucket, destConfigBucket, fs, destDB)
		if err != nil {
			fatalError(err)
		}
		fmt.Println("PIXLISE/PIQUANT Config files...")
		migrateConfigs(configBucket, destConfigBucket, fs) // fatalError called if any fail

		fmt.Println("Piquant Version...")
		err = migratePiquantVersion(configBucket, fs, destDB)
		if err != nil {
			fatalError(err)
		}
	}()

	/* Decided to leave PIQUANT configs in S3 because that way PIQUANT docker container has authenticated direct access
	fmt.Println("Piquant Configs...")
	err = migratePiquantConfigs(configBucket, fs, destDB)
	if err != nil {
		fatalError(err)
	}*/

	userContentPaths := []string{}
	go func() {
		defer listingWG.Done()
		// List all of S3 user contents
		fmt.Println("Listing user contents from S3...")
		var err error
		userContentPaths, err = fs.ListObjects(userContentBucket, filepaths.RootUserContent)
		if err != nil {
			fatalError(err)
		}
		fmt.Printf("  Listed %v files\n", len(userContentPaths))
	}()

	listingWG.Wait()

	fmt.Println("==========================================")
	fmt.Println("Migrating data from user content bucket...")
	fmt.Println("==========================================")

	var wg sync.WaitGroup

	if migrateZStacksEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Quant Z-stacks...")
			err = migrateMultiQuants(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of z-stacks...")
	}

	if migrateQuantsEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Quants...")
			err = migrateQuants(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, destUserContentBucket, userGroups)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of quants...")
	}

	if migrateElementSetsEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Element Sets...")
			err = migrateElementSets(userContentBucket, userContentPaths, fs, destDB, userGroups)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of element sets...")
	}

	if migrateROIsEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("ROIs...")
			err = migrateROIs(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, userGroups)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of ROIs...")
	}

	if migrateRGBMixesEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("RGB Mixes...")
			err = migrateRGBMixes(userContentBucket, userContentPaths, fs, destDB, userGroups)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of RGB mixes...")
	}

	if migrateTagsEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Tags...")
			err = migrateTags(userContentBucket, userContentPaths, fs, destDB)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of tags...")
	}

	if migrateDiffractionPeaksEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Diffraction Peak...")
			err = migrateDiffraction(userContentBucket, userContentPaths, fs, destDB)
			if err != nil {
				fatalError(err)
			}
		}()
	} else {
		fmt.Println("Skipping migration of diffraction peaks...")
	}

	// Wait for all
	wg.Wait()
	printFinishStats()
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

func clearOwnership(destDB *mongo.Database, objType protos.ObjectType) {
	coll := destDB.Collection(dbCollections.OwnershipName)

	result, err := coll.DeleteMany(context.TODO(), bson.M{"objecttype": objType})
	if err != nil {
		fatalError(err)
	}

	if result.DeletedCount <= 0 {
		log.Printf("Warning: Deleted 0 items from ownership collection for type: %v", objType)
	}
}

func fixROISharing(auth0Domain, auth0ClientId, auth0Secret string, fs fileaccess.S3Access, userContentBucket string, limitToDatasetIDsList []string, destDB *mongo.Database) {
	userGroups, err := migrateAuth0UserGroups(auth0Domain, auth0ClientId, auth0Secret, destDB)
	if err != nil {
		fatalError(err)
	}

	/*
		var listingWG sync.WaitGroup
		listingWG.Add(1)

		userContentPaths := []string{}
		go func() {
			defer listingWG.Done()
			// List all of S3 user contents
			fmt.Println("Listing user contents from S3...")
			var err error
			userContentPaths, err = fs.ListObjects(userContentBucket, filepaths.RootUserContent)
			if err != nil {
				fatalError(err)
			}
			fmt.Printf("  Listed %v files\n", len(userContentPaths))
		}()

		listingWG.Wait()


		paths := strings.Join(userContentPaths, "\n")
		err = os.WriteFile("roi-all-s3paths.txt", []byte(paths), 0777)
		if err != nil {
			fatalError(err)
		}
	*/

	paths, err := os.ReadFile("roi-all-s3paths.txt")
	if err != nil {
		fatalError(err)
	}

	userContentPaths := strings.Split(string(paths), "\n")

	err = migrateROIShares(userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, userGroups)
	if err != nil {
		fatalError(err)
	}
}

func runMigrateOrphanedSharedROIs(sharedIdsToMigrateFile string, fs fileaccess.S3Access, userContentBucket string, limitToDatasetIDsList []string, destDB *mongo.Database) {
	paths, err := os.ReadFile("roi-all-s3paths.txt")
	if err != nil {
		fatalError(err)
	}

	userContentPaths := strings.Split(string(paths), "\n")

	sharedIdsToMigrateFileContents, err := os.ReadFile(sharedIdsToMigrateFile)
	if err != nil {
		fatalError(err)
	}

	sharedIdsToMigrate := strings.Split(string(sharedIdsToMigrateFileContents), "\n")

	migrateOrphanedSharedROIs(sharedIdsToMigrate, userContentBucket, userContentPaths, limitToDatasetIDsList, fs, destDB, "m3l7kzikydo35znm")
}
