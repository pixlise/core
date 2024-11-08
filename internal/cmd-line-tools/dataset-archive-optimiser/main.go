package main

import (
	"archive/zip"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
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

var destMongoSecret string
var dbName string
var sourceDataBucket string
var destDataBucket string
var updateScript string

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	var mode string

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're importing to")
	flag.StringVar(&sourceDataBucket, "sourceDataBucket", "", "Data bucket so we can read archive zips")
	flag.StringVar(&destDataBucket, "destDataBucket", "", "Data bucket we're writing optimised archives to")
	flag.StringVar(&mode, "mode", "", "combine (to download zips and combine them into one) or update (to put the combined zips back to the source)")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		sourceDataBucket,
		destDataBucket,
	}
	checkNotEmptyName := []string{
		"sourceDataBucket",
		"destDataBucket",
	}
	if mode == "combine" {
		checkNotEmptyName = append(checkNotEmptyName, "dbName")
		checkNotEmpty = append(checkNotEmpty, dbName)
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

	if mode == "combine" {
		combine(sess, iLog, remoteFS, &localFS)
	} else if mode == "update" {
		update(remoteFS, &localFS)
	} else {
		fatalError(errors.New("Unknown mode: " + mode))
	}

	printFinishStats()
}

func update(remoteFS fileaccess.FileAccess, localFS fileaccess.FileAccess) {
	archivePrefix := "Archive/"
	archived, err := remoteFS.ListObjects(destDataBucket, archivePrefix)
	if err != nil {
		fatalError(err)
	}

	rttToArchiveFiles := map[string][]string{}
	for _, f := range archived {
		f = f[len(archivePrefix):]
		pos := strings.Index(f, "-")
		if pos > 0 {
			rtt := f[0:pos]
			if files, ok := rttToArchiveFiles[rtt]; !ok {
				rttToArchiveFiles[rtt] = []string{f}
			} else {
				files = append(files, f)
				rttToArchiveFiles[rtt] = files
			}
		}
	}

	// Loop through and delete any with just 1 zip file, we don't need to optimise those
	for rtt, files := range rttToArchiveFiles {
		if len(files) < 2 {
			delete(rttToArchiveFiles, rtt)
		}
	}

	// Handle each file
	for rtt, files := range rttToArchiveFiles {
		updateForRTT(rtt, files, remoteFS)
	}

	fmt.Println(updateScript)
}

func updateForRTT(rtt string, allZips []string, remoteFS fileaccess.FileAccess) {
	// See if we have any optimised files for this rtt
	archOptPrefix := "Archive-Optimised/"
	files, err := remoteFS.ListObjects(sourceDataBucket, archOptPrefix+rtt+"-")
	if err != nil {
		fmt.Printf("Failed to read archived files for rtt: %v, err: %v", rtt, err)
		return
	}

	if len(files) < 1 {
		fmt.Printf("No optimised archive for rtt: %v\n", rtt)
		return
	} else if len(files) > 1 {
		fmt.Printf("Too many optimised archive for rtt: %v\n", rtt)
		return
	}

	archivedZipName := files[0][len(archOptPrefix):]

	// Make sure the optimised file matches one in the list
	if !utils.ItemInSlice(archivedZipName, allZips) {
		fmt.Printf("Optimised archive name doesn't match any source files for: %v\n", rtt)
		return
	}

	// Delete all files older than or equal to this
	optimisedTS, err := getTimestamp(archivedZipName)
	if err != nil {
		fmt.Printf("Failed to read timestamp for: %v", archivedZipName)
		return
	}

	for _, f := range allZips {
		needUpdate := false
		if f == archivedZipName {
			needUpdate = true
		} else {
			ts, err := getTimestamp(f)
			if err != nil {
				fmt.Printf("Failed to read timestamp for: %v", f)
				return
			}
			needUpdate = ts < optimisedTS
		}

		if needUpdate {
			updateScript += fmt.Sprintf("aws s3 rm s3://%v/%v\n", destDataBucket, "Archive/"+f)
		}
	}

	// Copy the optimised file back
	updateScript += fmt.Sprintf("aws s3 cp s3://%v/%v s3://%v/%v\n", sourceDataBucket, "Archive-Optimised/"+archivedZipName, destDataBucket, "Archive/")
}

func getTimestamp(fileName string) (int, error) {
	_ /*expecting this to match already due to dir listing*/, timeStamp, err := datasetArchive.DecodeArchiveFileName(fileName)
	return timeStamp, err
}

func combine(sess *session.Session, iLog logger.ILogger, remoteFS fileaccess.FileAccess, localFS fileaccess.FileAccess) {
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
	if err != nil {
		log.Fatalln(err)
	}

	l := &logger.StdOutLogger{}

	skip := []string{}
	for _, scan := range scans {
		if utils.ItemInSlice(scan.Id, skip) {
			l.Infof("Skipping scan id: %v", scan.Id)
			continue
		}

		if scan.Instrument == protos.ScanInstrument_PIXL_FM {
			l.Infof("")
			l.Infof("============================================================")
			l.Infof(">>> Downloading archives for scan: %v (%v)", scan.Title, scan.Id)

			archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, localFS, l, sourceDataBucket, "" /* not needed */)
			_ /*localDownloadPath*/, localUnzippedPath, zipCount, lastZipName, err := archive.DownloadFromDatasetArchive(scan.Id, workingDir)
			if err != nil {
				log.Fatalf("Failed to download archive for scan %v: %v", scan.Id, err)
			}

			if zipCount == 1 {
				l.Infof("Only one zip was loaded, nothing to optimise...")
			}
			if zipCount <= 1 {
				// Stuff already logged... l.Infof("No archive zip files found for scan %v\n", scan.Id)
				continue
			}

			l.Infof("Zipping optimised archive %v...", lastZipName)

			// Now we zip up everything that's there
			//tm := time.Now()
			//zipName := fmt.Sprintf("%v-%02d-%02d-%v-%02d-%02d-%02d.zip", scan.Id, tm.Day(), int(tm.Month()), tm.Year(), tm.Hour(), tm.Minute(), tm.Second())
			// Zip with the latest zip name so if newer downlinks happened since we dont invent a time newer than them. This is to run on prod v3 but v4 has
			// been collecting its own archives for months. This also makes sense if we run against prodv4 in future.
			zipName := lastZipName
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
