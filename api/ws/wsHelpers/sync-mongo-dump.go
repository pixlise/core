package wsHelpers

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
)

var dataBackupLocalPath = "./backup"
var dataBackupS3Path = "DB"

func MakeMongoDumpInstance(mongoDetails mongoDBConnection.MongoConnectionDetails, dbName string) *mongodump.MongoDump {
	var toolOptions *options.ToolOptions

	ssl := options.SSL{}
	auth := options.Auth{
		Username: mongoDetails.User,
		Password: mongoDetails.Password,
	}

	connection := &options.Connection{
		Host: mongoDetails.Host,
		//Port: db.DefaultTestPort,
	}

	// Trim excess
	protocolPrefix := "mongodb://"
	connection.Host = strings.TrimPrefix(connection.Host, protocolPrefix)

	toolOptions = &options.ToolOptions{
		SSL:        &ssl,
		Connection: connection,
		Auth:       &auth,
		Verbosity:  &options.Verbosity{},
		URI:        &options.URI{},
	}

	toolOptions.Namespace = &options.Namespace{DB: dbName}

	outputOptions := &mongodump.OutputOptions{
		Out:  dataBackupLocalPath,
		Gzip: true,
		//Archive:                path.Join(dataBackupLocalPath, "archive.gzip"),
		NumParallelCollections: 1,
		//ExcludeCollections memoization??
	}
	inputOptions := &mongodump.InputOptions{}

	//log.SetVerbosity(toolOptions.Verbosity)

	return &mongodump.MongoDump{
		ToolOptions:   toolOptions,
		InputOptions:  inputOptions,
		OutputOptions: outputOptions,
	}
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
