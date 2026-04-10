package mongobackup

import (
	"archive/zip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
)

func BackupDB(dbName string, s3Bucket string, s3Path string, zipDBFiles bool, svcs *services.APIServices) error {
	startTimestamp := svcs.TimeStamper.GetTimeNowSec()

	localDBDumpDir := "./backup"
	zipPath := ""
	if zipDBFiles {
		zipPath = fmt.Sprintf("./%v %v.zip", dbName, time.Now().UTC().Format("02-Jan-2006 15-04-05"))
	}

	// Run MongoDump, save to a local archive file
	svcs.Log.Infof("DB Backup connecting...")

	dump, err := makeMongoDumpInstance(svcs.MongoConnectInfo, svcs.Log, dbName, localDBDumpDir)
	if err != nil {
		return fmt.Errorf("DB Bacup failed to create dump instance: %v", err)
	}

	err = dump.Init()
	if err != nil {
		return fmt.Errorf("DB Backup failed to initialise: %v", err)
	}

	// If we're not zipping, we want to clear the destination directory
	if !zipDBFiles {
		svcs.Log.Infof("Clearing previous DB backup files from: s3://%v/%v", s3Bucket, s3Path)

		err := fileaccess.ClearBucketDir(s3Bucket, s3Path, svcs.FS, svcs.Log)
		if err != nil {
			return fmt.Errorf("DB Backup file clearing failed: %v", err)
		}
	}

	svcs.Log.Infof("Starting DB dump...")
	err = dump.Dump()

	if err != nil {
		return fmt.Errorf("DB Backup dump generation failed: %v", err)
	}

	if zipDBFiles {
		svcs.Log.Infof("DB Dump complete, zipping files...")
		err = zipArchive(localDBDumpDir, zipPath)

		if err != nil {
			return fmt.Errorf("DB Backup failed to upload: %v", err)
		}

		svcs.Log.Infof("Uploading DB dump zip...")
		err = uploadFileToS3AsStream(zipPath, s3Bucket, s3Path, svcs)
	} else {
		svcs.Log.Infof("Uploading DB dump files...")
		err = uploadDirectoryToS3(localDBDumpDir, s3Bucket, s3Path, svcs)
	}

	if err != nil {
		return fmt.Errorf("DB Backup failed to upload: %v", err)
	}

	// Clean up the files generated so we don't keep filling the local drive
	svcs.Log.Infof("DB Backup deleting local db dump files...")
	err = os.RemoveAll(localDBDumpDir)
	if err != nil {
		svcs.Log.Errorf("Failed to clear local backup dir %v. Error: %v", localDBDumpDir, err)
	}

	if zipDBFiles {
		svcs.Log.Infof("DB Backup deleting local db dump zip file...")
		err = os.Remove(zipPath)
		if err != nil {
			svcs.Log.Errorf("Failed to clear local backup zip %v. Error: %v", zipPath, err)
		}
	}

	endTimestamp := svcs.TimeStamper.GetTimeNowSec()
	svcs.Log.Infof("DB Backup complete in %v sec", endTimestamp-startTimestamp)

	return nil
}

type LogWriter struct {
	logger logger.ILogger
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	w.logger.Infof("%s", string(p))
	return len(p), nil
}

func MakeMongoToolOptions(mongoDetails mongoDBConnection.MongoConnectionInfo, logger logger.ILogger, dbNamespace string) (*options.ToolOptions, error) {
	var toolOptions *options.ToolOptions

	log.SetVerbosity(nil /*toolOptions.Verbosity*/)
	lw := LogWriter{logger: logger}
	log.SetWriter(lw)

	ssl := options.SSL{}

	isLocal := strings.Contains(mongoDetails.Host, "localhost") && len(mongoDetails.Username) <= 0 && len(mongoDetails.Password) <= 0

	if !isLocal {
		ssl = options.SSL{
			UseSSL:        true,
			SSLCAFile:     "./global-bundle.pem",
			SSLPEMKeyFile: "./global-bundle.pem",
		}
	}

	auth := options.Auth{
		Username: mongoDetails.Username,
		Password: mongoDetails.Password,
	}

	connection := &options.Connection{
		Host: mongoDetails.Host,
	}

	// Trim excess
	protocolPrefix := "mongodb://"
	connection.Host = strings.TrimPrefix(connection.Host, protocolPrefix)

	connectionURI := fmt.Sprintf("mongodb://%s", connection.Host)

	uri, err := options.NewURI(connectionURI)
	if err != nil {
		return nil, err
	}

	retryWrites := false

	toolOptions = &options.ToolOptions{
		RetryWrites: &retryWrites,
		SSL:         &ssl,
		Connection:  connection,
		Auth:        &auth,
		Verbosity:   &options.Verbosity{},
		URI:         uri,
		Namespace:   &options.Namespace{DB: dbNamespace},
	}

	return toolOptions, nil
}

func makeMongoDumpInstance(mongoDetails mongoDBConnection.MongoConnectionInfo, logger logger.ILogger, dbName string, writePath string) (*mongodump.MongoDump, error) {
	toolOptions, err := MakeMongoToolOptions(mongoDetails, logger, dbName)
	if err != nil {
		return nil, err
	}

	logger.Infof("MongoDump connecting to: \"%v\", user: \"%v\", db-to-dump: \"%v\"...", toolOptions.URI.ConnectionString, toolOptions.Auth.Username, dbName)

	outputOptions := &mongodump.OutputOptions{
		Out:  writePath,
		Gzip: true,
		//Archive:                path.Join(dataBackupLocalPath, "archive.gzip"),
		NumParallelCollections: 1,
		//ExcludeCollections memoisation?? connecTokens??
	}
	inputOptions := &mongodump.InputOptions{}

	return &mongodump.MongoDump{
		ToolOptions:   toolOptions,
		InputOptions:  inputOptions,
		OutputOptions: outputOptions,
	}, nil
}

func uploadDirectoryToS3(dirPath string, s3Bucket string, s3Path string, svcs *services.APIServices) error {
	svcs.Log.Infof("uploadDirectoryToS3: Listing files in %v...", dirPath)
	localFS := fileaccess.FSAccess{}
	dbFiles, err := localFS.ListObjects(dirPath, "")
	if err != nil {
		return fmt.Errorf("Failed to list lfiles: %v", err)
	}

	for _, dbFile := range dbFiles {
		err = uploadFileToS3AsStream(filepath.Join(dirPath, dbFile), s3Bucket, s3Path, svcs)
		if err != nil {
			return err
		}
	}

	return nil
}

func uploadFileToS3AsStream(filePath string, s3Bucket string, s3Path string, svcs *services.APIServices) error {
	svcs.Log.Infof(" Uploading: %v...", filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("Failed to read local file: %v. Error: %v", filePath, err)
	}

	// Upload to bucket
	filePathRemote := path.Join(s3Path, filepath.Base(filePath))
	err = svcs.FS.WriteObjectStream(s3Bucket, filePathRemote, f)

	if err != nil {
		return fmt.Errorf("Failed to upload file: %v. Error: %v", filePathRemote, err)
	}

	//svcs.Log.Infof("  OK")
	return nil
}

func zipArchive(dirPath string, outputZip string) error {
	zipFile, err := os.Create(outputZip)
	if err != nil {
		return fmt.Errorf("Failed to create zip output file: %v. Error: %v", outputZip, err)
	}

	zipWriter := zip.NewWriter(zipFile)

	err = utils.AddFilesToZip(zipWriter, dirPath, "")
	if err != nil {
		return fmt.Errorf("Failed to create zip %v for directory %v: %v", outputZip, dirPath, err)
	}

	err = zipWriter.Close()
	if err != nil {
		return fmt.Errorf("Failed to close written zip %v for directory %v: %v", outputZip, dirPath, err)
	}

	return nil
}
