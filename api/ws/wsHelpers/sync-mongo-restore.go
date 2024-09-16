package wsHelpers

import (
	"fmt"
	"path"
	"strings"

	"github.com/mongodb/mongo-tools/common/log"
	"github.com/mongodb/mongo-tools/common/options"
	"github.com/mongodb/mongo-tools/mongorestore"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
)

func MakeMongoRestoreInstance(mongoDetails mongoDBConnection.MongoConnectionDetails, logger logger.ILogger, restoreToDBName string, restoreFromDBName string) (*mongorestore.MongoRestore, error) {
	var toolOptions *options.ToolOptions

	ssl := options.SSL{
		UseSSL:        true,
		SSLCAFile:     "./global-bundle.pem",
		SSLPEMKeyFile: "./global-bundle.pem",
	}

	auth := options.Auth{
		Username: mongoDetails.User,
		Password: mongoDetails.Password,
	}

	connection := &options.Connection{
		Host: mongoDetails.Host,
	}

	// Trim excess
	protocolPrefix := "mongodb://"
	connection.Host = strings.TrimPrefix(connection.Host, protocolPrefix)

	connectionURI := fmt.Sprintf("mongodb://%s/%s", connection.Host, "")

	logger.Infof("MongoRestore connecting to: %v, user %v, restore-to-db: %v, restore-from-db: %v...", connectionURI, auth.Username, restoreToDBName, restoreFromDBName)

	uri, err := options.NewURI(connectionURI)
	if err != nil {
		logger.Errorf("%v", err)
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
	}

	toolOptions.Namespace = &options.Namespace{DB: restoreToDBName}

	outputOptions := &mongorestore.OutputOptions{
		NumParallelCollections: 1,
		NumInsertionWorkers:    1,
		Drop:                   true,
		NoIndexRestore:         true,
	}

	inputOptions := &mongorestore.InputOptions{
		Gzip: true,
		//Archive:                path.Join(dataBackupLocalPath, "archive.gzip"),
		Directory:              "./" + path.Join(dataBackupLocalPath, restoreFromDBName),
		RestoreDBUsersAndRoles: false,
	}

	nsOptions := &mongorestore.NSOptions{
		NSInclude: []string{"*"},
		NSFrom:    []string{restoreFromDBName},
		NSTo:      []string{restoreToDBName},
	}

	log.SetVerbosity(nil /*toolOptions.Verbosity*/)
	lw := LogWriter{logger: logger}
	log.SetWriter(lw)

	return mongorestore.New(mongorestore.Options{
		ToolOptions:     toolOptions,
		InputOptions:    inputOptions,
		OutputOptions:   outputOptions,
		NSOptions:       nsOptions,
		TargetDirectory: inputOptions.Directory,
	})
}

type LogWriter struct {
	logger logger.ILogger
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	w.logger.Infof(string(p))
	return len(p), nil
}

func DownloadArchive(svcs *services.APIServices) (string, error) {
	svcs.Log.Infof("Downloading PIXLISE DB Dump files...")

	remoteDBFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, dataBackupS3Path)
	if err != nil {
		return "", fmt.Errorf("Failed to list remote DB dump files: %v", err)
	}

	svcs.Log.Infof("Found %v remote DB Dump files...", len(remoteDBFiles))

	localFS := fileaccess.FSAccess{}

	dbName := ""
	for _, dbFile := range remoteDBFiles {
		svcs.Log.Infof(" Downloading: %v...", dbFile)

		dbFileBytes, err := svcs.FS.ReadObject(svcs.Config.DataBackupBucket, dbFile)
		if err != nil {
			return "", fmt.Errorf("Failed to download remote DB dump file: %v. Error: %v", dbFile, err)
		}

		// Save locally
		// Remove remote root dir
		dbFilePathLocal := strings.TrimPrefix(dbFile, dataBackupS3Path+"/")
		err = localFS.WriteObject(dataBackupLocalPath, dbFilePathLocal, dbFileBytes)

		if err != nil {
			return "", fmt.Errorf("Failed to write local DB dump file: %v. Error: %v", dbFilePathLocal, err)
		}

		// Save the first dir as the db name
		if len(dbName) <= 0 {
			parts := strings.Split(dbFile, "/")
			if len(parts) > 1 {
				dbName = parts[1]
			}
		}
	}

	svcs.Log.Infof("PIXLISE DB Dump files downloaded")
	return dbName, nil
}
