package wsHelpers

import (
	"fmt"
	"strings"

	"github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongorestore"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
)

func MakeMongoRestoreInstance(mongoDetails mongoDBConnection.MongoConnectionDetails, dbName string) (*mongorestore.MongoRestore, error) {
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

	outputOptions := &mongorestore.OutputOptions{
		NumParallelCollections: 1,
		NumInsertionWorkers:    1,
		Drop:                   true,
		NoIndexRestore:         true,
	}

	inputOptions := &mongorestore.InputOptions{
		Gzip: true,
		//Archive:                path.Join(dataBackupLocalPath, "archive.gzip"),
		Directory:              "./backup/pixlise-prodv4-15-jul-2024", //"./" + path.Join(dataBackupLocalPath, dbName),
		RestoreDBUsersAndRoles: false,
	}

	nsOptions := &mongorestore.NSOptions{
		NSInclude: []string{"*"},
		//NSInclude: []string{"pixlise-prodv4-15-jul-2024.ownership"},
		NSFrom: []string{"pixlise-prodv4-15-jul-2024"},
		NSTo:   []string{"pixlise-localdev"},
	}

	//log.SetVerbosity(toolOptions.Verbosity)

	return mongorestore.New(mongorestore.Options{
		ToolOptions:     toolOptions,
		InputOptions:    inputOptions,
		OutputOptions:   outputOptions,
		NSOptions:       nsOptions,
		TargetDirectory: inputOptions.Directory,
	})
}

func DownloadArchive(svcs *services.APIServices) error {
	svcs.Log.Infof("Downloading PIXLISE DB Dump files...")

	remoteDBFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, dataBackupS3Path)
	if err != nil {
		return fmt.Errorf("Failed to list remote DB dump files: %v", err)
	}

	svcs.Log.Infof("Found %v remote DB Dump files...", len(remoteDBFiles))

	localFS := fileaccess.FSAccess{}

	for _, dbFile := range remoteDBFiles {
		svcs.Log.Infof(" Downloading: %v...", dbFile)

		dbFileBytes, err := svcs.FS.ReadObject(svcs.Config.DataBackupBucket, dbFile)
		if err != nil {
			return fmt.Errorf("Failed to download remote DB dump file: %v. Error: %v", dbFile, err)
		}

		// Save locally
		// Remove remote root dir
		dbFilePathLocal := strings.TrimPrefix(dbFile, dataBackupS3Path+"/")
		err = localFS.WriteObject(dataBackupLocalPath, dbFilePathLocal, dbFileBytes)

		if err != nil {
			return fmt.Errorf("Failed to write local DB dump file: %v. Error: %v", dbFilePathLocal, err)
		}
	}

	svcs.Log.Infof("PIXLISE DB Dump files downloaded")
	return nil
}
