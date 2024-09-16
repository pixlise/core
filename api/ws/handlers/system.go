package wsHandler

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mongodb/mongo-tools/mongodump"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleBackupDBReq(req *protos.BackupDBReq, hctx wsHelpers.HandlerContext) (*protos.BackupDBResp, error) {
	if len(hctx.Svcs.Config.DataBackupBucket) <= 0 {
		err := "PIXLISE Backup bucket not configured"
		hctx.Svcs.Log.Errorf(err)
		return nil, errors.New(err)
	}

	if !hctx.Svcs.Config.BackupEnabled {
		err := "PIXLISE Backup not enabled"
		hctx.Svcs.Log.Errorf(err)
		return nil, errors.New(err)
	}

	// Delete any local backups already done so we don't run out of space/have issues with overwriting
	err := wsHelpers.ResetLocalMongoBackupDir()
	if err != nil {
		return nil, err
	}

	startTimestamp := hctx.Svcs.TimeStamper.GetTimeNowSec()

	hctx.Svcs.Log.Infof("PIXLISE Backup Requested, will be written to bucket: %v", hctx.Svcs.Config.DataBackupBucket)

	// Run MongoDump, save to a local archive file
	dump := wsHelpers.MakeMongoDumpInstance(hctx.Svcs.MongoDetails, hctx.Svcs.Log, mongoDBConnection.GetDatabaseName("pixlise", hctx.Svcs.Config.EnvironmentName))

	err = dump.Init()
	if err != nil {
		hctx.Svcs.Log.Errorf("PIXLISE Backup failed to initialise: %v", err)
		return nil, err
	}

	go runBackup(dump, startTimestamp, hctx.Svcs)

	return &protos.BackupDBResp{}, nil
}

func clearBucket(bucket string, fs fileaccess.FileAccess, logger logger.ILogger) error {
	files, err := fs.ListObjects(bucket, "")
	if err != nil {
		return err
	}

	logger.Infof("Clearing %v files from bucket: %v", len(files), bucket)

	for c, file := range files {
		logger.Infof("Clearing file %v of %v...", c, len(files))

		err = fs.DeleteObject(bucket, file)
		if err != nil {
			return err
		}
	}

	logger.Infof("Bucket cleared: %v", bucket)
	return nil
}

func runBackup(dump *mongodump.MongoDump, startTimestamp int64, svcs *services.APIServices) {
	var wg sync.WaitGroup
	var errDBDump error
	var errScanSync error
	var errImageSync error
	var errQuantSync error

	svcs.Log.Infof("Clearing PIXLISE backup bucket: %v", svcs.Config.DataBackupBucket)

	err := clearBucket(svcs.Config.DataBackupBucket, svcs.FS, svcs.Log)
	if err != nil {
		svcs.Log.Errorf("PIXLISE Backup bucket clear failed: %v", err)
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		errDBDump = dump.Dump()

		if errDBDump == nil {
			svcs.Log.Infof("PIXLISE DB Dump complete")
			errDBDump = wsHelpers.UploadArchive(svcs)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Syncing scans to bucket")
		errScanSync = wsHelpers.SyncScans(svcs)
		svcs.Log.Infof("Syncing scans to bucket COMPLETE")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Syncing quants to bucket")
		errQuantSync = wsHelpers.SyncQuants(svcs)
		svcs.Log.Infof("Syncing quants to bucket COMPLETE")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Syncing images to bucket")
		errImageSync = wsHelpers.SyncImages(svcs)
		svcs.Log.Infof("Syncing images to bucket COMPLETE")
	}()

	// Wait for all sync tasks
	wg.Wait()

	if errDBDump != nil {
		err = fmt.Errorf("PIXLISE Backup DB dump failed: %v", errDBDump)
	}

	if errScanSync != nil {
		err = fmt.Errorf("PIXLISE Backup error syncing scans: %v", errScanSync)
	}

	if errImageSync != nil {
		err = fmt.Errorf("PIXLISE Backup error syncing images: %v", errImageSync)
	}

	if errQuantSync != nil {
		err = fmt.Errorf("PIXLISE Backup error syncing quants: %v", errQuantSync)
	}

	if err != nil {
		svcs.Log.Errorf("%v", err)
		return
	}

	endTimestamp := svcs.TimeStamper.GetTimeNowSec()
	svcs.Log.Infof("PIXLISE Backup complete in %v sec", endTimestamp-startTimestamp)

	// TODO: send an update message to notify anything listening that we're done!
}

func HandleRestoreDBReq(req *protos.RestoreDBReq, hctx wsHelpers.HandlerContext) (*protos.RestoreDBResp, error) {
	// Only allow restore if enabled and we're NOT prod
	if !hctx.Svcs.Config.RestoreEnabled {
		err := "PIXLISE Restore not enabled"
		hctx.Svcs.Log.Errorf(err)
		return nil, errors.New(err)
	}

	if strings.Contains(strings.ToLower(hctx.Svcs.Config.EnvironmentName), "prod") {
		err := "PIXLISE Restore not allowed on environment: " + hctx.Svcs.Config.EnvironmentName
		hctx.Svcs.Log.Errorf(err)
		return nil, errors.New(err)
	}

	deleteLocal := true

	if deleteLocal {
		// Delete any local backups already done so we don't run out of space/have issues with overwriting
		err := wsHelpers.ResetLocalMongoBackupDir()
		if err != nil {
			return nil, err
		}
	}

	startTimestamp := hctx.Svcs.TimeStamper.GetTimeNowSec()

	go runRestore(startTimestamp, hctx.Svcs, deleteLocal)

	return &protos.RestoreDBResp{}, nil
}

func runRestore(startTimestamp int64, svcs *services.APIServices, downloadRemoteFiles bool) {
	var wg sync.WaitGroup
	var errDBRestore error
	var errScanSync error
	var errImageSync error
	var errQuantSync error

	wg.Add(1)
	go func() {
		defer wg.Done()

		restoreFromDBName := ""
		if downloadRemoteFiles {
			restoreFromDBName, errDBRestore = wsHelpers.DownloadArchive(svcs)
		}

		if errDBRestore == nil {
			restore, errDBRestore := wsHelpers.MakeMongoRestoreInstance(svcs.MongoDetails, svcs.Log, mongoDBConnection.GetDatabaseName("pixlise", svcs.Config.EnvironmentName), restoreFromDBName)

			if errDBRestore == nil {
				result := restore.Restore()
				if result.Err != nil {
					errDBRestore = result.Err
				} else {
					svcs.Log.Infof("Mongo Restore complete: %v successes, %v failures", result.Successes, result.Failures)

					if downloadRemoteFiles {
						// Delete the local db archive
						errDBRestore = wsHelpers.ClearLocalMongoArchive()
					}
				}
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Restoring scans to bucket")
		errScanSync = wsHelpers.RestoreScans(svcs)
		svcs.Log.Infof("Restoring scans to bucket COMPLETE")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Restoring quants to bucket")
		errQuantSync = wsHelpers.RestoreQuants(svcs)
		svcs.Log.Infof("Restoring quants to bucket COMPLETE")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		svcs.Log.Infof("Restoring images to bucket")
		errImageSync = wsHelpers.RestoreImages(svcs)
		svcs.Log.Infof("Restoring images to bucket COMPLETE")
	}()

	// Wait for all sync tasks
	wg.Wait()

	var err error
	if errDBRestore != nil {
		err = fmt.Errorf("PIXLISE Restore DB restore failed: %v", errDBRestore)
	}

	if errScanSync != nil {
		err = fmt.Errorf("PIXLISE Restore error restoring scans: %v", errScanSync)
	}

	if errImageSync != nil {
		err = fmt.Errorf("PIXLISE Restore error restoring images: %v", errImageSync)
	}

	if errQuantSync != nil {
		err = fmt.Errorf("PIXLISE Restore error restoring quants: %v", errQuantSync)
	}

	if err != nil {
		return
	}

	endTimestamp := svcs.TimeStamper.GetTimeNowSec()
	svcs.Log.Infof("PIXLISE Restore complete in %v sec", endTimestamp-startTimestamp)

	// TODO: send an update message to notify anything listening that we're done!
}

func HandleDBAdminConfigGetReq(req *protos.DBAdminConfigGetReq, hctx wsHelpers.HandlerContext) (*protos.DBAdminConfigGetResp, error) {
	// Reply depending on how this env is configured
	resp := protos.DBAdminConfigGetResp{
		CanBackup:          len(hctx.Svcs.Config.DataBackupBucket) > 0 && hctx.Svcs.Config.BackupEnabled,
		BackupDestination:  hctx.Svcs.Config.DataBackupBucket,
		CanRestore:         len(hctx.Svcs.Config.DataBackupBucket) > 0 && hctx.Svcs.Config.RestoreEnabled,
		RestoreFrom:        hctx.Svcs.Config.DataBackupBucket,
		ImpersonateEnabled: hctx.Svcs.Config.ImpersonateEnabled,
	}

	return &resp, nil
}
