package wsHandler

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getScanFileById(ctx context.Context, id string, hctx wsHelpers.HandlerContext) (*protos.ScanFile, error) {
	if err := wsHelpers.CheckStringField(&id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	result := hctx.Svcs.MongoDB.Collection(dbCollections.ScanFilesName).FindOne(ctx, bson.M{"_id": id})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(id)
		}
		return nil, result.Err()
	}

	file := &protos.ScanFile{}
	if err := result.Decode(file); err != nil {
		return nil, err
	}
	return file, nil
}

func HandleScanFileListReq(req *protos.ScanFileListReq, hctx wsHelpers.HandlerContext) (*protos.ScanFileListResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Anyone who can view the scan can see its files
	if _, err := wsHelpers.CheckObjectAccess(false, req.ScanId, protos.ObjectType_OT_SCAN, hctx); err != nil {
		return nil, err
	}

	files, err := wsHelpers.ListScanFiles(context.TODO(), req.ScanId, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		wsHelpers.PrepareScanFileForClient(file, hctx.Svcs)
	}

	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return &protos.ScanFileListResp{Files: files}, nil
}

func HandleScanFileWriteReq(req *protos.ScanFileWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScanFileWriteResp, error) {
	ctx := context.TODO()

	req.Name = strings.TrimSpace(req.Name)
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, wsHelpers.ScanFileNameMaxLength); err != nil {
		return nil, err
	}
	if strings.ContainsAny(req.Name, "/\\") {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Name cannot contain / or \\"))
	}
	if err := wsHelpers.CheckStringField(&req.Description, "Description", 0, wsHelpers.ScanFileDescriptionMaxLength); err != nil {
		return nil, err
	}

	file, err := getScanFileById(ctx, req.Id, hctx)
	if err != nil {
		return nil, err
	}

	if _, err := wsHelpers.CheckObjectAccess(true, file.ScanId, protos.ObjectType_OT_SCAN, hctx); err != nil {
		return nil, err
	}

	if req.Name != file.Name {
		exists, err := wsHelpers.ScanFileNameExists(ctx, file.ScanId, req.Name, file.Id, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("A file named \"%v\" already exists for this scan", req.Name))
		}
	}

	file.Name = req.Name
	file.Description = req.Description
	file.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())

	update := bson.D{
		{Key: "name", Value: file.Name},
		{Key: "description", Value: file.Description},
		{Key: "modifiedunixsec", Value: file.ModifiedUnixSec},
	}

	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ScanFilesName).UpdateByID(ctx, file.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}
	if result.MatchedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.Id)
	}

	wsHelpers.PrepareScanFileForClient(file, hctx.Svcs)
	return &protos.ScanFileWriteResp{File: file}, nil
}

func HandleScanFileDeleteReq(req *protos.ScanFileDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ScanFileDeleteResp, error) {
	ctx := context.TODO()

	file, err := getScanFileById(ctx, req.Id, hctx)
	if err != nil {
		return nil, err
	}

	if _, err := wsHelpers.CheckObjectAccess(true, file.ScanId, protos.ObjectType_OT_SCAN, hctx); err != nil {
		return nil, err
	}

	fs, bucket := wsHelpers.GetScanFileStore(hctx.Svcs)
	if err := fs.DeleteObject(bucket, file.StoragePath); err != nil && !fs.IsNotFoundError(err) {
		return nil, fmt.Errorf("Failed to delete file \"%v\": %v", file.Name, err)
	}

	_, err = hctx.Svcs.MongoDB.Collection(dbCollections.ScanFilesName).DeleteOne(ctx, bson.M{"_id": file.Id})
	if err != nil {
		return nil, err
	}

	return &protos.ScanFileDeleteResp{}, nil
}
