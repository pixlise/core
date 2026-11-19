package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const ScanFileNameMaxLength = 255
const ScanFileDescriptionMaxLength = 2000

// Returns where files attached to scans are stored
func GetScanFileStore(svcs *services.APIServices) (fileaccess.FileAccess, string) {
	fs := svcs.ScanFilesFS
	if fs == nil {
		fs = svcs.FS
	}

	bucket := svcs.Config.ScanFilesBucket
	if len(bucket) <= 0 {
		bucket = svcs.Config.DatasetsBucket
	}

	return fs, bucket
}

func ListScanFiles(ctx context.Context, scanId string, db *mongo.Database) ([]*protos.ScanFile, error) {
	cursor, err := db.Collection(dbCollections.ScanFilesName).Find(ctx, bson.M{"scanid": scanId})
	if err != nil {
		return nil, err
	}

	files := []*protos.ScanFile{}
	err = cursor.All(ctx, &files)
	return files, err
}

func ScanFileNameExists(ctx context.Context, scanId string, name string, ignoreFileId string, db *mongo.Database) (bool, error) {
	filter := bson.M{"scanid": scanId, "name": name}
	if len(ignoreFileId) > 0 {
		filter["_id"] = bson.M{"$ne": ignoreFileId}
	}

	count, err := db.Collection(dbCollections.ScanFilesName).CountDocuments(ctx, filter)
	return count > 0, err
}

// Fills the fields that are only sent to clients, and clears the ones that should not be
func PrepareScanFileForClient(file *protos.ScanFile, svcs *services.APIServices) {
	file.StoragePath = ""

	if len(file.CreatorUserId) > 0 {
		if user, err := getUserInfo(file.CreatorUserId, svcs.MongoDB, svcs.TimeStamper); err == nil && user != nil {
			file.Creator = &protos.UserInfo{Id: user.Id, Name: user.Name, IconURL: user.IconURL}
		}
	}
}

// Deletes all files attached to a scan, both from storage and DB
func DeleteAllScanFiles(ctx context.Context, scanId string, svcs *services.APIServices) error {
	files, err := ListScanFiles(ctx, scanId, svcs.MongoDB)
	if err != nil {
		return err
	}

	fs, bucket := GetScanFileStore(svcs)

	failed := 0
	for _, file := range files {
		if err := fs.DeleteObject(bucket, file.StoragePath); err != nil && !fs.IsNotFoundError(err) {
			svcs.Log.Errorf("Failed to delete file %v (%v) attached to scan %v: %v", file.Id, file.StoragePath, scanId, err)
			failed++
		}
	}

	_, err = svcs.MongoDB.Collection(dbCollections.ScanFilesName).DeleteMany(ctx, bson.M{"scanid": scanId})
	if err != nil {
		return err
	}

	if failed > 0 {
		return fmt.Errorf("%v of %v files attached to scan %v failed to delete from storage", failed, len(files), scanId)
	}
	return nil
}
