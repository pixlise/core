package wsHelpers

import (
	"fmt"
	"os"
	"path"

	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
)

var dataBackupLocalPath = "./backup"
var dataBackupS3Path = "DB"

func MakeMongoDumpInstance(mongoDetails mongoDBConnection.MongoConnectionDetails, logger logger.ILogger, dbName string) (*mongodump.MongoDump, error) {
	toolOptions, err := makeMongoToolOptions(mongoDetails, logger, dbName)
	if err != nil {
		return nil, err
	}

	logger.Infof("MongoDump connecting to: %v, user %v, db-to-dump: %v...", toolOptions.URI.ConnectionString, toolOptions.Auth.Username, dbName)

	outputOptions := &mongodump.OutputOptions{
		Out:  dataBackupLocalPath,
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

func UploadArchive(svcs *services.APIServices) error {
	svcs.Log.Infof("UploadArchive: Reading PIXLISE DB Dump files...")

	localFS := fileaccess.FSAccess{}
	dbFiles, err := localFS.ListObjects(dataBackupLocalPath, "")
	if err != nil {
		return fmt.Errorf("Failed to list local DB dump files: %v", err)
	}

	svcs.Log.Infof("Found %v DB Dump files...", len(dbFiles))

	for _, dbFile := range dbFiles {
		svcs.Log.Infof(" Uploading: %v...", dbFile)

		dbFilePath := path.Join(dataBackupLocalPath, dbFile)
		dbFileBytes, err := os.ReadFile(dbFilePath)
		if err != nil {
			return fmt.Errorf("Failed to read local DB dump file: %v. Error: %v", dbFilePath, err)
		}

		// Upload to bucket
		dbFilePathRemote := path.Join(dataBackupS3Path, dbFile)
		err = svcs.FS.WriteObject(svcs.Config.DataBackupBucket, dbFilePathRemote, dbFileBytes)

		if err != nil {
			return fmt.Errorf("Failed to upload DB dump file: %v. Error: %v", dbFilePathRemote, err)
		}
	}

	svcs.Log.Infof("PIXLISE DB Dump write complete")
	return nil
}
