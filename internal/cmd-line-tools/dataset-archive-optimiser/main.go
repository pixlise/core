package main

import (
	"archive/zip"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pixlise/core/v4/api/dataimport/datasetArchive"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var t0 = time.Now().UnixMilli()

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var destMongoSecret string
	var dbName string
	var sourceDataBucket string
	var destDataBucket string

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're importing to")
	flag.StringVar(&sourceDataBucket, "sourceDataBucket", "", "Data bucket so we can read archive zips")
	flag.StringVar(&destDataBucket, "destDataBucket", "", "Data bucket we're writing optimised archives to")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		sourceDataBucket,
		destDataBucket,
		dbName,
	}
	checkNotEmptyName := []string{
		"sourceDataBucket",
		"destDataBucket",
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

	remoteFS := fileaccess.MakeS3Access(s3svc)
	localFS := fileaccess.FSAccess{}

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

	cursor, err := coll.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		log.Fatalln(err)
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(ctx, &scans)
	if err != nil {
		log.Fatalln(err)
	}

	// Loop through all scans and read each archive one-by-one
	workingDir, err := os.MkdirTemp("", "archive-fix-")

	l := &logger.StdOutLogger{}

	for _, scan := range scans {
		if scan.Instrument == protos.ScanInstrument_PIXL_FM {
			l.Infof("")
			l.Infof("============================================================")
			l.Infof(">>> Downloading archives for scan: %v (%v)", scan.Title, scan.Id)

			archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, &localFS, l, sourceDataBucket, "" /* not needed */)
			_ /*localDownloadPath*/, localUnzippedPath, zipCount, err := archive.DownloadFromDatasetArchive(scan.Id, workingDir)
			if err != nil {
				log.Fatalf("Failed to download archive for scan %v: %v", scan.Id, err)
			}

			if zipCount == 0 {
				// Stuff already logged... l.Infof("No archive zip files found for scan %v\n", scan.Id)
				continue
			}

			l.Infof("Zipping optimised archive...")

			// Now we zip up everything that's there
			tm := time.Now()
			zipName := fmt.Sprintf("%v-%02d-%02d-%v-%02d-%02d-%02d.zip", scan.Id, tm.Day(), int(tm.Month()), tm.Year(), tm.Hour(), tm.Minute(), tm.Second())
			zipPath := filepath.Join(workingDir, zipName)
			zipFile, err := os.Create(zipPath)
			if err != nil {
				log.Fatalf("Failed to create zip output file for scan %v: %v", scan.Id, err)
			}

			zipWriter := zip.NewWriter(zipFile)
			/*_, err = zipWriter.Create(zipPath)
			if err != nil {
				log.Fatalf("Failed to create zip output file for scan %v: %v", scan.Id, err)
			}*/

			err = utils.AddFilesToZip(zipWriter, localUnzippedPath, "")
			if err != nil {
				log.Fatalf("Failed to create optimised zip %v for scan %v: %v", zipPath, scan.Id, err)
			}

			err = zipWriter.Close()
			if err != nil {
				log.Fatalf("Failed to close written zip %v for scan %v: %v", zipPath, scan.Id, err)
			}

			// Upload the zip to S3
			uploadPath := path.Join("Archive-Optimised", zipName)
			l.Infof("Uploading optimised archive to s3://%v/%v", destDataBucket, uploadPath)

			zipData, err := os.ReadFile(zipPath)
			if err != nil {
				log.Fatalf("Failed to read created zip output file %v for scan %v: %v", zipPath, scan.Id, err)
			}

			if len(zipData) <= 0 {
				l.Infof("Created optimized zip archive %v for scan %v was 0 bytes, skipping upload\n", zipPath, scan.Id)
				continue
			}

			err = remoteFS.WriteObject(destDataBucket, uploadPath, zipData)
			if err != nil {
				log.Fatalf("Failed to upload zip output file for scan %v: %v", scan.Id, err)
			}

			// Delete all downloaded files and created zip
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
