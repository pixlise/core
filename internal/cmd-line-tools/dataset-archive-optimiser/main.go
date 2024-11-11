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
var dataBucket string

func main() {
	fmt.Printf("Started: %v\n", time.Now().String())

	flag.StringVar(&destMongoSecret, "destMongoSecret", "", "Destination mongo DB secret")
	flag.StringVar(&dbName, "dbName", "", "DB name we're importing to")
	flag.StringVar(&dataBucket, "dataBucket", "", "Data bucket so we can read archive zips and write back optimised zips")

	flag.Parse()

	// Check they're not empty
	checkNotEmpty := []string{
		dataBucket,
		dbName,
	}
	checkNotEmptyName := []string{
		"dataBucket",
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

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	rtts := readRTTs(sess, iLog)
	optimise(rtts, &remoteFS, iLog)

	printFinishStats()
}

func readRTTs(sess *session.Session, iLog logger.ILogger) map[string]string {
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
		fatalError(err)
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(ctx, &scans)
	if err != nil {
		fatalError(err)
	}

	rtts := map[string]string{}
	skip := []string{}
	for _, scan := range scans {
		if utils.ItemInSlice(scan.Id, skip) {
			iLog.Infof("Skipping scan id: %v", scan.Id)
			continue
		}

		if scan.Instrument == protos.ScanInstrument_PIXL_FM {
			rtts[scan.Id] = scan.Title
		}
	}

	return rtts
}

func optimise(rtts map[string]string, remoteFS fileaccess.FileAccess, iLog logger.ILogger) {
	archivePrefix := "Archive/"
	archived, err := remoteFS.ListObjects(dataBucket, archivePrefix)
	if err != nil {
		fatalError(err)
	}

	rttToArchiveFiles := map[string][]string{}
	for _, f := range archived {
		f = f[len(archivePrefix):]
		pos := strings.Index(f, "-")
		if pos > 0 {
			rtt := f[0:pos]

			// Make sure it's one of the ones we're interested in optimising (it's for an actual PIXL scan)
			if _, ok := rtts[rtt]; ok {
				if files, ok := rttToArchiveFiles[rtt]; !ok {
					rttToArchiveFiles[rtt] = []string{f}
				} else {
					files = append(files, f)
					rttToArchiveFiles[rtt] = files
				}
			}
		}
	}

	// Loop through and delete any with just 1 zip file, we don't need to optimise those
	for rtt, files := range rttToArchiveFiles {
		if len(files) < 2 {
			delete(rttToArchiveFiles, rtt)
		}
	}

	// Loop through all scans and read each archive one-by-one
	workingDir, err := os.MkdirTemp("", "archive-fix-")
	if err != nil {
		fatalError(err)
	}

	iLog.Infof("Found %v scans to optimise archive for:", len(rttToArchiveFiles))
	for rtt, files := range rttToArchiveFiles {
		iLog.Infof(" %v: %v zip files", rtt, len(files))
	}

	// Handle each file
	for rtt, _ := range rttToArchiveFiles {
		localArchivePath, zipFilesOptimised, err := makeOptimisedArchive(rtt, rtts[rtt], remoteFS, workingDir, iLog)
		if err != nil {
			iLog.Errorf("Error creating optimised archive for %v: %v\n", rtt, err)
		} else {
			// Upload the optimised file (should be overwriting the latest one)
			err = upload(localArchivePath, "Archive", remoteFS, iLog)
			if err != nil {
				iLog.Errorf("FAILED TO UPLOAD archive file %v: %v\n", localArchivePath, err)
			}

			// Delete the zips that we are replacing
			for _, zipFile := range zipFilesOptimised {
				// Don't delete what we just uploaded!
				if !strings.HasSuffix(localArchivePath, zipFile) {
					zipPath := path.Join("Archive", zipFile)

					iLog.Infof("Deleting from S3: %v", zip.ErrInsecurePath)
					err = remoteFS.DeleteObject(dataBucket, zipPath)
					if err != nil {
						iLog.Errorf("Error deleting archive file %v: %v\n", zipPath, err)
					}
				}
			}
		}
	}
}

func makeOptimisedArchive(rtt string, scanTitle string, remoteFS fileaccess.FileAccess, workingDir string, iLog logger.ILogger) (string, []string, error) {
	l := &logger.StdOutLogger{}
	localFS := fileaccess.FSAccess{}

	l.Infof("")
	l.Infof("============================================================")
	l.Infof(">>> Downloading archives for scan: %v (%v)", scanTitle, rtt)

	archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, &localFS, l, dataBucket, "" /* not needed */)
	_ /*localDownloadPath*/, localUnzippedPath, zipFilesOrdered, err := archive.DownloadFromDatasetArchive(rtt, workingDir)
	if err != nil {
		return "", []string{}, fmt.Errorf("Failed to download archive for scan %v: %v", rtt, err)
	}

	zipCount := len(zipFilesOrdered)
	if zipCount == 1 {
		l.Infof("Only one zip was loaded, nothing to optimise...")
	}
	if zipCount <= 1 {
		// Stuff already logged... l.Infof("No archive zip files found for scan %v\n", scan.Id)
		return "", zipFilesOrdered, nil
	}

	lastZipName := zipFilesOrdered[len(zipFilesOrdered)-1]

	// Remove any paths...
	lastZipName = filepath.Base(lastZipName)
	for c, p := range zipFilesOrdered {
		zipFilesOrdered[c] = filepath.Base(p)
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
		return "", zipFilesOrdered, fmt.Errorf("Failed to create zip output file for scan %v: %v", rtt, err)
	}

	zipWriter := zip.NewWriter(zipFile)
	/*_, err = zipWriter.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("Failed to create zip output file for scan %v: %v", scan.Id, err)
	}*/

	err = utils.AddFilesToZip(zipWriter, localUnzippedPath, "")
	if err != nil {
		return "", zipFilesOrdered, fmt.Errorf("Failed to create optimised zip %v for scan %v: %v", zipPath, rtt, err)
	}

	err = zipWriter.Close()
	if err != nil {
		return "", zipFilesOrdered, fmt.Errorf("Failed to close written zip %v for scan %v: %v", zipPath, rtt, err)
	}

	return zipPath, zipFilesOrdered, nil
}

func upload(localArchivePath string, remotePath string, remoteFS fileaccess.FileAccess, iLog logger.ILogger) error {
	zipName := filepath.Base(localArchivePath)

	// Upload the zip to S3
	uploadPath := path.Join("Archive", zipName)
	iLog.Infof("Uploading optimised archive to s3://%v/%v", dataBucket, uploadPath)

	zipData, err := os.ReadFile(localArchivePath)
	if err != nil {
		return fmt.Errorf("Failed to read created zip output file %v: %v", localArchivePath, err)
	}

	if len(zipData) <= 0 {
		iLog.Infof("Created optimized zip archive %v was 0 bytes, skipping upload\n", localArchivePath)
		return nil
	}

	return remoteFS.WriteObject(dataBucket, uploadPath, zipData)
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
