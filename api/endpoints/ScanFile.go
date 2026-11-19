package endpoints

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const ScanFileIdentifier = "fileid"

const MaxScanFileSizeBytes = 250 * 1024 * 1024

func checkScanAccessForHttpUser(ctx context.Context, requireEdit bool, scanId string, svcs *services.APIServices, requestingUserId string) (string, error) {
	userId, err := checkImpersonation(ctx, svcs, requestingUserId)
	if err != nil {
		return "", fmt.Errorf("Failed to determine user id impersonation status: %v", err)
	}

	memberOfGroupIds, isMemberOfNoGroups := wsHelpers.GetCachedUserGroupMembership(userId)
	viewerOfGroupIds, isViewerOfNoGroups := wsHelpers.GetCachedUserGroupViewership(userId)
	if !isMemberOfNoGroups && !isViewerOfNoGroups {
		return "", errorwithstatus.MakeBadRequestError(errors.New("User has no group membership, can't determine permissions"))
	}

	_, err = wsHelpers.CheckObjectAccessForUser(requireEdit, scanId, protos.ObjectType_OT_SCAN, userId, memberOfGroupIds, viewerOfGroupIds, svcs.MongoDB)
	return userId, err
}

func getScanFileContentType(fileName string, requestContentType string) string {
	if byExt := mime.TypeByExtension(strings.ToLower(path.Ext(fileName))); len(byExt) > 0 {
		return byExt
	}

	if mediaType, _, err := mime.ParseMediaType(requestContentType); err == nil && len(mediaType) > 0 {
		return mediaType
	}

	return "application/octet-stream"
}

func PutScanFile(params apiRouter.ApiHandlerGenericParams) error {
	ctx := params.Request.Context()

	if !params.UserInfo.Permissions["EDIT_SCAN"] {
		return errorwithstatus.MakeBadRequestError(errors.New("PutScanFile not allowed"))
	}

	scanId := params.PathParams[ScanIdentifier]
	if err := wsHelpers.CheckStringField(&scanId, "scan", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return err
	}

	fileName := strings.ReplaceAll(params.PathParams[FileNameIdentifier], "\\", "/")
	fileName = strings.TrimSpace(path.Base(fileName))
	if fileName == "." || fileName == ".." || fileName == "/" {
		fileName = ""
	}
	if err := wsHelpers.CheckStringField(&fileName, "filename", 1, wsHelpers.ScanFileNameMaxLength); err != nil {
		return err
	}

	if params.Request.ContentLength > MaxScanFileSizeBytes {
		return errorwithstatus.MakeStatusError(http.StatusRequestEntityTooLarge, fmt.Errorf("File is too large, maximum size is %v MB", MaxScanFileSizeBytes/1024/1024))
	}

	userId, err := checkScanAccessForHttpUser(ctx, true, scanId, params.Svcs, params.UserInfo.UserID)
	if err != nil {
		return err
	}

	exists, err := wsHelpers.ScanFileNameExists(ctx, scanId, fileName, "", params.Svcs.MongoDB)
	if err != nil {
		return err
	}
	if exists {
		return errorwithstatus.MakeStatusError(http.StatusConflict, fmt.Errorf("A file named \"%v\" already exists for this scan", fileName))
	}

	fs, bucket := wsHelpers.GetScanFileStore(params.Svcs)

	fileId := params.Svcs.IDGen.GenObjectID()
	storagePath := filepaths.GetScanAttachedFilePath(scanId, fileId, fileaccess.MakeValidObjectName(fileName, true))

	body := &io.LimitedReader{R: params.Request.Body, N: MaxScanFileSizeBytes + 1}
	if err := fs.WriteObjectStream(bucket, storagePath, body); err != nil {
		return fmt.Errorf("Failed to store file \"%v\": %v", fileName, err)
	}

	_, sizeBytes, err := fs.ObjectExists(bucket, storagePath)
	if err == nil {
		if sizeBytes <= 0 {
			err = errorwithstatus.MakeBadRequestError(fmt.Errorf("File \"%v\" is empty", fileName))
		} else if sizeBytes > MaxScanFileSizeBytes {
			err = errorwithstatus.MakeStatusError(http.StatusRequestEntityTooLarge, fmt.Errorf("File is too large, maximum size is %v MB", MaxScanFileSizeBytes/1024/1024))
		}
	}

	now := uint32(params.Svcs.TimeStamper.GetTimeNowSec())
	file := &protos.ScanFile{
		Id:              fileId,
		ScanId:          scanId,
		Name:            fileName,
		ContentType:     getScanFileContentType(fileName, params.Request.Header.Get("Content-Type")),
		SizeBytes:       uint64(sizeBytes),
		CreatorUserId:   userId,
		CreatedUnixSec:  now,
		ModifiedUnixSec: now,
		StoragePath:     storagePath,
	}

	if err == nil {
		_, err = params.Svcs.MongoDB.Collection(dbCollections.ScanFilesName).InsertOne(ctx, file)
	}

	if err != nil {
		if delErr := fs.DeleteObject(bucket, storagePath); delErr != nil {
			params.Svcs.Log.Errorf("PutScanFile: Failed to clean up %v/%v: %v", bucket, storagePath, delErr)
		}
		return err
	}

	params.Svcs.Log.Infof("PutScanFile: Stored \"%v\" (%v bytes) for scan %v at: %v/%v", fileName, sizeBytes, scanId, bucket, storagePath)

	wsHelpers.PrepareScanFileForClient(file, params.Svcs)
	utils.SendProtoJSON(params.Writer, file)
	return nil
}

func GetScanFile(params apiRouter.ApiHandlerGenericParams) error {
	ctx := params.Request.Context()

	scanId := params.PathParams[ScanIdentifier]
	fileId := params.PathParams[ScanFileIdentifier]

	if _, err := checkScanAccessForHttpUser(ctx, false, scanId, params.Svcs, params.UserInfo.UserID); err != nil {
		return err
	}

	result := params.Svcs.MongoDB.Collection(dbCollections.ScanFilesName).FindOne(ctx, bson.M{"_id": fileId, "scanid": scanId})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return errorwithstatus.MakeNotFoundError(fileId)
		}
		return result.Err()
	}

	file := &protos.ScanFile{}
	if err := result.Decode(file); err != nil {
		return err
	}

	fs, bucket := wsHelpers.GetScanFileStore(params.Svcs)

	exists, sizeBytes, err := fs.ObjectExists(bucket, file.StoragePath)
	if err != nil {
		return err
	}
	if !exists {
		return errorwithstatus.MakeNotFoundError(file.Name)
	}

	stream, err := fs.ReadObjectStream(bucket, file.StoragePath)
	if err != nil {
		if fs.IsNotFoundError(err) {
			return errorwithstatus.MakeNotFoundError(file.Name)
		}
		return err
	}
	defer stream.Close()

	header := params.Writer.Header()
	header.Set("Content-Type", "application/octet-stream")
	header.Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%v", url.PathEscape(file.Name)))
	header.Set("Content-Length", fmt.Sprintf("%v", sizeBytes))
	header.Set("Cache-Control", "private, no-cache")
	header.Set("X-Content-Type-Options", "nosniff")

	if _, err := io.Copy(params.Writer, stream); err != nil {
		return fmt.Errorf("Error copying file \"%v\" to the http response: %v", file.Name, err)
	}

	return nil
}
