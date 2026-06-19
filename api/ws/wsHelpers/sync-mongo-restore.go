package wsHelpers

import (
	"fmt"
	"path"
	"strings"

	"github.com/mongodb/mongo-tools/mongorestore"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/mongobackup"
	"github.com/pixlise/core/v4/core/utils"
)

func MakeMongoRestoreInstance(mongoDetails mongoDBConnection.MongoConnectionInfo, logger logger.ILogger, restoreToDBName string, restoreFromDBName string) (*mongorestore.MongoRestore, error) {
	toolOptions, err := mongobackup.MakeMongoToolOptions(mongoDetails, logger, restoreToDBName)
	if err != nil {
		return nil, err
	}

	logger.Infof("MongoRestore connecting to: %v, user %v, restore-to-db: %v, restore-from-db: %v...", toolOptions.URI.ConnectionString, mongoDetails.Username, restoreToDBName, restoreFromDBName)

	outputOptions := &mongorestore.OutputOptions{
		NumParallelCollections: 1,
		NumInsertionWorkers:    1,
		Drop:                   true,
		NoIndexRestore:         true,
	}

	inputOptions := &mongorestore.InputOptions{
		Gzip: true,
		//Archive:                path.Join(dataBackupLocalPath, "archive.gzip"),
		Directory:              "./" + path.Join("backup", restoreFromDBName),
		RestoreDBUsersAndRoles: false,
	}

	nsOptions := &mongorestore.NSOptions{
		NSInclude: []string{"*"},
		NSFrom:    []string{restoreFromDBName},
		NSTo:      []string{restoreToDBName},
	}

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
	w.logger.Infof("%s", string(p))
	return len(p), nil
}

func DownloadArchive(envS3Path string, svcs *services.APIServices) (string, error) {
	svcs.Log.Infof("Downloading PIXLISE DB Dump files...")

	dataBackupS3Path := path.Join(envS3Path, "DB")
	remoteDBFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, dataBackupS3Path)
	if err != nil {
		return "", fmt.Errorf("Failed to list remote DB dump files: %v", err)
	}

	svcs.Log.Infof("Found %v remote DB Dump files...", len(remoteDBFiles))

	localFS := fileaccess.FSAccess{}
	dbName := ""

	for _, dbFile := range remoteDBFiles {
		// Check if it's one to exclude
		isExcluded := false
		for _, excludeColl := range svcs.Config.RestoreExcludeCollections {
			if strings.Contains(dbFile, excludeColl) {
				svcs.Log.Infof(" Skipping restore of collection %v as it containing RestoreExcludeCollections item %v", dbFile, excludeColl)
				isExcluded = true
				break
			}
		}

		if isExcluded {
			continue
		}

		// Report free space remaining
		freeBytes, err := utils.GetDiskAvailableBytes()
		if err != nil {
			svcs.Log.Errorf(" Failed to get free disk bytes: %v", err)
		}

		// Remove remote root dir
		dbFilePathLocal := strings.TrimPrefix(dbFile, dataBackupS3Path+"/")

		svcs.Log.Infof(" Downloading: %v -> %v... (%v bytes free)", dbFile, "./backup/"+dbFilePathLocal, freeBytes)

		dbStream, err := svcs.FS.ReadObjectStream(svcs.Config.DataBackupBucket, dbFile)
		if err != nil {
			return "", fmt.Errorf("Failed to download remote DB dump file: %v. Error: %v", dbFile, err)
		}

		// Save locally
		err = localFS.WriteObjectStream("./backup", dbFilePathLocal, dbStream)

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

	svcs.Log.Infof("PIXLISE DB Dump files downloaded, db name: %v", dbName)
	return dbName, nil
}
