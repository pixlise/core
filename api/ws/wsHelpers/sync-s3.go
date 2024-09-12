package wsHelpers

import (
	"context"
	"path"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SyncScans(svcs *services.APIServices) error {
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.ScansName)

	filter := bson.D{}
	opts := options.Find().SetProjection(bson.M{"_id": true})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return err
	}

	scanIds := []bson.M{}
	err = cursor.All(context.TODO(), &scanIds)
	if err != nil {
		return err
	}

	scanPaths := []string{}
	for _, item := range scanIds {
		id := item["_id"].(string)
		scanPaths = append(scanPaths, path.Join(id, "dataset.bin"))
		scanPaths = append(scanPaths, path.Join(id, "diffraction-db.bin"))
	}

	return syncFiles(svcs.Config.DatasetsBucket, filepaths.DatasetScansRoot, scanPaths, svcs.Config.DataBackupBucket, filepaths.DatasetScansRoot, svcs.FS, svcs.Log)
}

func SyncImages(svcs *services.APIServices) error {
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.ImagesName)

	filter := bson.D{}
	opts := options.Find().SetProjection(bson.M{"_id": true})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return err
	}

	items := []bson.M{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return err
	}

	imagePaths := []string{}
	for _, item := range items {
		imagePaths = append(imagePaths, item["_id"].(string))
	}

	return syncFiles(svcs.Config.DatasetsBucket, filepaths.DatasetImagesRoot, imagePaths, svcs.Config.DataBackupBucket, filepaths.DatasetImagesRoot, svcs.FS, svcs.Log)
}

func SyncQuants(svcs *services.APIServices) error {
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.QuantificationsName)

	filter := bson.D{}
	opts := options.Find().SetProjection(bson.M{"status.outputfilepath": true, "_id": true})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return err
	}

	quantPaths := []string{}

	for cursor.Next(ctx) {
		type QuantStatus struct {
			OutputFilePath string
		}
		type QuantItem struct {
			Id     string `bson:"_id"`
			Status QuantStatus
		}

		item := QuantItem{}
		err = cursor.Decode(&item)
		if err != nil {
			return err
		}

		// Copying bin and csv files
		p := path.Join(item.Status.OutputFilePath, item.Id)

		quantPaths = append(quantPaths, p+".bin")
		quantPaths = append(quantPaths, p+".csv")
	}

	return syncFiles(svcs.Config.UsersBucket, "" /*filepaths.RootQuantificationPath*/, quantPaths, svcs.Config.DataBackupBucket, filepaths.RootQuantificationPath, svcs.FS, svcs.Log)
}

func RestoreScans(svcs *services.APIServices) error {
	// List source files
	scanFiles, err := svcs.FS.ListObjects(svcs.Config.DatasetsBucket, filepaths.DatasetScansRoot)
	if err != nil {
		return err
	}

	return syncFiles(svcs.Config.DataBackupBucket, filepaths.DatasetScansRoot, scanFiles, svcs.Config.DatasetsBucket, filepaths.DatasetScansRoot, svcs.FS, svcs.Log)
}

func RestoreQuants(svcs *services.APIServices) error {
	// List source files
	quantFiles, err := svcs.FS.ListObjects(svcs.Config.UsersBucket, filepaths.RootQuantificationPath)
	if err != nil {
		return err
	}

	return syncFiles(
		svcs.Config.DataBackupBucket,
		path.Join(filepaths.RootQuantificationPath, filepaths.RootQuantificationPath),
		quantFiles,
		svcs.Config.UsersBucket,
		filepaths.RootQuantificationPath,
		svcs.FS,
		svcs.Log)
}

func RestoreImages(svcs *services.APIServices) error {
	// List source files
	imageFiles, err := svcs.FS.ListObjects(svcs.Config.DatasetsBucket, filepaths.DatasetImagesRoot)
	if err != nil {
		return err
	}

	return syncFiles(svcs.Config.DataBackupBucket, filepaths.DatasetImagesRoot, imageFiles, svcs.Config.DatasetsBucket, filepaths.DatasetImagesRoot, svcs.FS, svcs.Log)
}

func syncFiles(srcBucket string, srcRoot string, filePaths []string, destBucket string, destRoot string, fs fileaccess.FileAccess, log logger.ILogger) error {
	// Get a listing from the destination
	destFiles, err := fs.ListObjects(destBucket, destRoot)
	if err != nil {
		return err
	}

	// Form a map, so we can see what's missing quickly
	destFileLookup := map[string]bool{}
	for _, f := range destFiles {
		// Snip off the root so the comparison is valid
		if strings.HasPrefix(f, destRoot+"/") {
			f = f[len(destRoot)+1:]
		}

		destFileLookup[f] = true
	}

	// Find which files haven't yet been copied across
	toCopy := []string{}
	for _, f := range filePaths {
		if !destFileLookup[f] {
			toCopy = append(toCopy, f)
		}
	}

	// Copy all the files
	for c, f := range toCopy {
		if c%100 == 0 {
			log.Infof(" Sync backup directory to %v: %v of %v copied...", destRoot, c, len(toCopy))
		}

		srcPath := path.Join(srcRoot, f)
		err = fs.CopyObject(srcBucket, srcPath, destBucket, path.Join(destRoot, f))
		if err != nil {
			if fs.IsNotFoundError(err) {
				log.Errorf(" Sync backup source file not found: s3://%v/%v", srcBucket, srcPath)
			} else {
				log.Errorf(" Sync error reading read s3://%v/%v: %v", srcBucket, srcPath, err)
			}
			//return err
		}
	}

	return nil
}
