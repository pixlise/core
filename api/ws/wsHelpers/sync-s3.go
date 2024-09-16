package wsHelpers

import (
	"context"
	"fmt"
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

	relativeQuantFiles, err := makeRelativePaths(quantPaths, filepaths.RootQuantificationPath)
	if err != nil {
		return err
	}

	return syncFiles(svcs.Config.UsersBucket, filepaths.RootQuantificationPath, relativeQuantFiles, svcs.Config.DataBackupBucket, filepaths.RootQuantificationPath, svcs.FS, svcs.Log)
}

func RestoreScans(svcs *services.APIServices) error {
	// List source files
	scanFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, filepaths.DatasetScansRoot)
	if err != nil {
		return err
	}

	// Make scans relative!
	relativeScanFiles, err := makeRelativePaths(scanFiles, filepaths.DatasetScansRoot)
	if err != nil {
		return err
	}

	return syncFiles(svcs.Config.DataBackupBucket, filepaths.DatasetScansRoot, relativeScanFiles, svcs.Config.DatasetsBucket, filepaths.DatasetScansRoot, svcs.FS, svcs.Log)
}

func RestoreQuants(svcs *services.APIServices) error {
	// List source files
	quantFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, filepaths.RootQuantificationPath)
	if err != nil {
		return err
	}

	relativeQuantFiles, err := makeRelativePaths(quantFiles, filepaths.RootQuantificationPath)
	if err != nil {
		return err
	}

	return syncFiles(
		svcs.Config.DataBackupBucket,
		filepaths.RootQuantificationPath,
		relativeQuantFiles,
		svcs.Config.UsersBucket,
		filepaths.RootQuantificationPath,
		svcs.FS,
		svcs.Log)
}

func RestoreImages(svcs *services.APIServices) error {
	// List source files
	imageFiles, err := svcs.FS.ListObjects(svcs.Config.DataBackupBucket, filepaths.DatasetImagesRoot)
	if err != nil {
		return err
	}

	relativeImageFiles, err := makeRelativePaths(imageFiles, filepaths.DatasetImagesRoot)
	if err != nil {
		return err
	}

	return syncFiles(svcs.Config.DataBackupBucket, filepaths.DatasetImagesRoot, relativeImageFiles, svcs.Config.DatasetsBucket, filepaths.DatasetImagesRoot, svcs.FS, svcs.Log)
}

func makeRelativePaths(fullPaths []string, root string) ([]string, error) {
	relativeFiles := []string{}
	for _, fullPath := range fullPaths {
		if !strings.HasPrefix(fullPath, root+"/") {
			return []string{}, fmt.Errorf("Path: %v is not relative to %v", fullPath, root)
		}

		relativeFiles = append(relativeFiles, fullPath[len(root)+1:])
	}

	return relativeFiles, nil
}

// Requires source bucket, source root with srcRelativePaths which are relative to source root. This way the relative paths
// can be compared to dest paths, and only the ones not already in dest root are copied.
func syncFiles(srcBucket string, srcRoot string, srcRelativePaths []string, destBucket string, destRoot string, fs fileaccess.FileAccess, log logger.ILogger) error {
	// Get a listing from the destination
	// NOTE: the returned paths contain destRoot at the start!
	destFullFiles, err := fs.ListObjects(destBucket, destRoot)
	if err != nil {
		return err
	}

	// Form a map, so we can see what's missing quickly
	destFileLookup := map[string]bool{}
	for _, f := range destFullFiles {
		// Store dest file without root, so we can compare
		if strings.HasPrefix(f, destRoot+"/") {
			f = f[len(destRoot)+1:]
		}

		destFileLookup[f] = true
	}

	// Find which files haven't yet been copied across
	toCopyRelativePaths := []string{}
	for _, srcRelative := range srcRelativePaths {
		if !destFileLookup[srcRelative] {
			toCopyRelativePaths = append(toCopyRelativePaths, srcRelative)
		}
	}

	log.Infof(" Sync backup directory to %v: %v skipped (already at destination)...", destRoot, len(srcRelativePaths)-len(toCopyRelativePaths))

	// Copy all the files
	for c, relSrcPath := range toCopyRelativePaths {
		if c%100 == 0 {
			log.Infof(" Sync backup directory to %v: %v of %v copied...", destRoot, c, len(toCopyRelativePaths))
		}

		srcFullPath := path.Join(srcRoot, relSrcPath)
		err = fs.CopyObject(srcBucket, srcFullPath, destBucket, path.Join(destRoot, relSrcPath))
		if err != nil {
			if fs.IsNotFoundError(err) {
				log.Errorf(" Sync backup source file not found: s3://%v/%v", srcBucket, srcFullPath)
			} else {
				log.Errorf(" Sync error reading read s3://%v/%v: %v", srcBucket, srcFullPath, err)
			}
			//return err
		}
	}

	return nil
}
