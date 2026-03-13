// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package endpoints

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/imageedit"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/vips"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

const ScanIdentifier = "scan"
const FileNameIdentifier = "filename"

func getBoolValue(str string) (bool, error) {
	if len(str) <= 0 {
		return false, nil
	}

	switch str {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %v", str)
	}
}

func addFileNameSuffix(name string, suffix string) string {
	ext := path.Ext(name)
	name = name[0 : len(name)-len(ext)]
	return name + suffix + ext
}

func GetImage(params apiRouter.ApiHandlerStreamParams) (*s3.GetObjectOutput, string, string, string, int, error) {
	// Path elements
	scanID := params.PathParams[ScanIdentifier]
	requestedFileName := path.Join(scanID, params.PathParams[FileNameIdentifier])

	// User might be impersonating someone, check this
	userId, err := checkImpersonation(params.Svcs, params.UserInfo.UserID)
	if err != nil {
		return nil, "", "", "", 0, fmt.Errorf("Failed to determine user id impersonation status: %v", err)
	}

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, isMemberOfNoGroups := wsHelpers.GetCachedUserGroupMembership(userId)
	viewerOfGroupIds, isViewerOfNoGroups := wsHelpers.GetCachedUserGroupViewership(userId)
	if !isMemberOfNoGroups && !isViewerOfNoGroups {
		// User is probably not logged in
		return nil, "", "", "", 0, errorwithstatus.MakeBadRequestError(errors.New("User has no group membership, can't determine permissions"))
	}

	// Now read the DB record for the image, so we can determine what scans it's associated with
	ctx := context.TODO()
	coll := params.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	filter := wsHelpers.GetDBImageFilter(requestedFileName)
	cursor, err := coll.Find(ctx, filter)

	if err != nil {
		// This doesn't look good...
		if err == mongo.ErrNoDocuments {
			return nil, "", "", "", 0, errorwithstatus.MakeNotFoundError(requestedFileName)
		}
		return nil, "", "", "", 0, err
	}

	scanImages := []*protos.ScanImage{}
	err = cursor.All(context.TODO(), &scanImages)
	if err != nil {
		return nil, "", "", "", 0, err
	}

	dbImages, err := wsHelpers.GetLatestImagesOnly(scanImages)
	if err != nil {
		return nil, "", "", "", 0, err
	}

	if len(dbImages) != 1 {
		return nil, "", "", "", 0, fmt.Errorf("Failed to find image %v, version count was %v", requestedFileName, len(dbImages))
	}

	dbImage := dbImages[0]

	for _, scanId := range dbImage.AssociatedScanIds {
		_, err := wsHelpers.CheckObjectAccessForUser(false, scanId, protos.ObjectType_OT_SCAN, userId, memberOfGroupIds, viewerOfGroupIds, params.Svcs.MongoDB)
		if err != nil {
			return nil, "", "", "", 0, err
		}
	}

	// We're still here, so we have access! Check query params for any modifiers
	finalFileName := dbImage.ImagePath

	// If we have any of the tile fields in the request, we need all of them!
	tileFields := []string{"layer", "tilex", "tiley"}
	tileFieldValues := map[string]int{}

	for _, field := range tileFields {
		if f, ok := params.PathParams[field]; ok {
			v, err := strconv.ParseInt(f, 10, 0)
			if err != nil {
				return nil, "", "", "", 0, fmt.Errorf("Failed to read %v: %v", field, err)
			}

			if v < 0 {
				return nil, "", "", "", 0, fmt.Errorf("Invalid input for field %v: %v", field, v)
			}

			tileFieldValues[field] = int(v)
		}
	}

	if len(tileFieldValues) > 0 {
		// We're expecting tile fields, check that we got them all
		if len(tileFieldValues) != len(tileFields) {
			return nil, "", "", "", 0, fmt.Errorf("Not all tile lookup fields were provided. Expected: [%v]", strings.Join(tileFields, ","))
		}

		// Check they're valid
		layer := tileFieldValues["layer"]
		tilex := tileFieldValues["tilex"]
		tiley := tileFieldValues["tiley"]

		suffix, err := getTilePathSuffix(params.Svcs.MongoDB, dbImage, layer, tilex, tiley)
		if err != nil {
			return nil, "", "", "", 0, fmt.Errorf("Invalid tile requested: %v", err)
		}

		finalFileName = path.Join(finalFileName, suffix)
	} else {
		// User didn't request a specific tile, but if it's a pyramid, we read the top level image
		finalFileName = addPyramidPathIfNeeded(finalFileName, dbImage)
	}

	showLocations, err := getBoolValue(params.PathParams["with-locations"])
	if err != nil {
		return nil, "", "", "", 0, err
	} else if showLocations {
		if len(tileFieldValues) > 0 {
			return nil, "", "", "", 0, fmt.Errorf("Cannot provide locations on image tile")
		}

		finalFileName = addFileNameSuffix(finalFileName, "-withloc")
	}

	var minWidthPx = 0
	if minWStr, ok := params.PathParams["minwidth"]; ok {
		if len(tileFieldValues) > 0 {
			return nil, "", "", "", 0, fmt.Errorf("Cannot downscale image tile")
		}

		// We DO have a value set for this, read it
		minWidthPx, err = strconv.Atoi(minWStr)
		if err != nil {
			return nil, "", "", "", 0, err
		} else {
			// We got a min width, round it to our step size
			if minWidthPx < 0 {
				minWidthPx = 0
			}
			step := minWidthPx / imageSizeStepPx

			if step <= 0 {
				step = 1
			}

			minWidthPx = imageSizeStepPx * step

			// Only do this if we're not scaling it up!
			if minWidthPx < int(dbImage.Width) {
				finalFileName = addFileNameSuffix(finalFileName, fmt.Sprintf("-width%v", minWidthPx))
			} else {
				minWidthPx = 0 // pretend user didn't ask to scale it
			}
		}
	}

	statuscode := 200

	// Load from dataset directory unless custom loading is requested, where we look up the file in the manual bucket
	imgBucket := params.Svcs.Config.DatasetsBucket

	// Check if the file exists, as we may be able to generate a version of the underlying file to satisfy the request
	var s3Path string
	if minWidthPx > 0 || showLocations {
		s3Path = filepaths.GetImageCacheFilePath(finalFileName)
	} else {
		s3Path = filepaths.GetImageFilePath(finalFileName)
	}

	_, err = params.Svcs.S3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(imgBucket),
		Key:    aws.String(s3Path),
	})
	if err != nil {
		if minWidthPx > 0 || showLocations {
			// If the file doesn't exist, check if the base file name exists, because we may just need to generate
			// a modified version of it
			imageGenFrom := addPyramidPathIfNeeded(requestedFileName, dbImage)
			genS3Path := filepaths.GetImageFilePath(imageGenFrom)

			// Original file exists, generate this modified copy and cache it back in S3 for the rest of this
			// function to find!
			err = generateImageVersion(imageGenFrom, genS3Path, minWidthPx, showLocations, s3Path, params.Svcs)
		}

		if err != nil {
			// Failed at this, stop here
			return nil, "", "", "", 0, err
		}
	}

	if params.Headers != nil && params.Headers.Get("If-None-Match") != "" {
		head, err := params.Svcs.S3.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(imgBucket),
			Key:    aws.String(s3Path),
		})
		if err == nil {
			if head != nil && head.ETag != nil {
				header := params.Headers.Get("If-None-Match")
				if header != "" && strings.Contains(header, *head.ETag) {
					statuscode = http.StatusNotModified
					return nil, requestedFileName, *head.ETag, head.LastModified.String(), statuscode, nil
				}
			}
		}
	}

	if params.Headers != nil && params.Headers.Get("If-Modified-Since") != "" {
		head, err := params.Svcs.S3.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(imgBucket),
			Key:    aws.String(s3Path),
		})
		if err == nil {
			if head != nil && head.LastModified != nil {
				header := params.Headers.Get("If-Modified-Since")
				if header != "" && strings.Contains(header, head.LastModified.String()) {
					statuscode = http.StatusNotModified
					return nil, requestedFileName, *head.ETag, head.LastModified.String(), statuscode, nil
				}
			}
		}
	}

	obj := &s3.GetObjectInput{
		Bucket: aws.String(imgBucket),
		Key:    aws.String(s3Path),
	}

	result, err := params.Svcs.S3.GetObject(obj)
	var etag = ""
	var lm = time.Time{}
	if result != nil && result.ETag != nil {
		params.Svcs.Log.Debugf("ETAG for cache: %s, s3://%v/%v", *result.ETag, imgBucket, s3Path)
		etag = *result.ETag
	}

	if result != nil && result.LastModified != nil {
		lm = *result.LastModified
		params.Svcs.Log.Debugf("Last Modified for cache: %v, s3://%v/%v", lm, imgBucket, s3Path)
	}

	params.Svcs.Log.Debugf("Image GET: s3://%v/%v", imgBucket, s3Path)
	return result, requestedFileName, etag, lm.String(), 0, err
}

func addPyramidPathIfNeeded(imagePath string, image *protos.ScanImage) string {
	if len(image.PyramidId) > 0 {
		imagePath = path.Join(imagePath, "0", "0_0."+image.PyramidTileFormat)
	}
	return imagePath
}

func getTilePathSuffix(db *mongo.Database, pyramidImage *protos.ScanImage, layer, tileX, tileY int) (string, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagePyramidsName)

	filter := bson.M{"_id": pyramidImage.PyramidId}
	pyramidResult := coll.FindOne(ctx, filter)
	if pyramidResult.Err() != nil {
		return "", pyramidResult.Err()
	}

	pyramid := &protos.ImagePyramidDBEntry{}
	err := pyramidResult.Decode(pyramid)
	if err != nil {
		return "", err
	}

	if pyramid.Id != pyramidImage.PyramidId {
		return "", fmt.Errorf("Unexpected image pyramid id: %v in pyramid %v", pyramid.Id, pyramidImage.PyramidId)
	}

	if layer < 0 || layer >= len(pyramid.Pyramid.Pyramid) {
		return "", fmt.Errorf("Invalid pyramid level: %v", layer)
	}

	pyramidLevel := pyramid.Pyramid.Pyramid[layer]

	idx := tileY*int(pyramidLevel.TilesWide) + tileX
	if idx >= len(pyramidLevel.Tiles) {
		return "", fmt.Errorf("Tile x: %v, y: %v for pyramid level: %v does not exist", tileX, tileY, layer)
	}

	suffix := path.Join(strconv.Itoa(layer), fmt.Sprintf("%v_%v.%v", tileX, tileY, pyramidImage.PyramidTileFormat))
	return suffix, nil
}

const imageSizeStepPx = 200

func generateImageVersion(imageName string, s3Path string, minWidthPx int, showLocations bool, finalFilePath string, svcs *services.APIServices) error {
	if minWidthPx <= 0 {
		return fmt.Errorf("generateImageVersion minWidthPx too small: %v", minWidthPx)
	}
	imgBytes, err := svcs.FS.ReadObject(svcs.Config.DatasetsBucket, s3Path)
	if err != nil {
		return err
	}

	img, imgFormat, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return err
	}

	// Apply the mods
	if showLocations {
		// Read locations from DB for this image
		ctx := context.TODO()
		coll := svcs.MongoDB.Collection(dbCollections.ImageBeamLocationsName)

		filter := bson.M{"_id": dataImportHelpers.GetImageNameSansVersion(imageName)}
		result := coll.FindOne(ctx, filter)

		if result.Err() != nil {
			// We don't stop here, we just log the error and allow the image generation to complete
			svcs.Log.Errorf("Failed to load beam locations for marking on image %v. Error was: %v", s3Path, result.Err())
		} else {
			locs := protos.ImageLocations{}
			err = result.Decode(&locs)
			if err != nil {
				svcs.Log.Errorf("Failed to decode loaded beam locations for marking on image %v. Error was: %v", s3Path, result.Err())
			} else {
				// Merge all coordinates
				coords := []*protos.Coordinate2D{}
				for _, coordsForScan := range locs.LocationPerScan {
					coords = append(coords, coordsForScan.Locations...)
				}
				// Mark them on the image
				img = imageedit.MarkLocations(img, coords, color.Black, nil)
			}
		}
	}

	if minWidthPx > 0 {
		img = imageedit.ScaleImage(img, minWidthPx)
	}

	// Now we're done, read the bytes out in the right format
	imgBytesOut, err := imageedit.GetImageBytes(img, imgFormat)
	if err != nil {
		return err
	}

	// Write the final image to final cached destination
	return svcs.FS.WriteObject(svcs.Config.DatasetsBucket, finalFilePath, imgBytesOut)
}

type FilePartRecvItem struct {
	LastPartNo uint32
	TotalParts uint32
	BytesSoFar uint64
}

var filePartsRecvd map[string]FilePartRecvItem = map[string]FilePartRecvItem{}

func PutImage(params apiRouter.ApiHandlerGenericParams) error {
	if !params.UserInfo.Permissions["EDIT_SCAN"] {
		return errorwithstatus.MakeBadRequestError(errors.New("PutImage not allowed"))
	}

	// User might be impersonating someone, check this
	userId, err := checkImpersonation(params.Svcs, params.UserInfo.UserID)
	if err != nil {
		return fmt.Errorf("Failed to determine user id impersonation status: %v", err)
	}

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, isMemberOfNoGroups := wsHelpers.GetCachedUserGroupMembership(userId)
	viewerOfGroupIds, isViewerOfNoGroups := wsHelpers.GetCachedUserGroupViewership(userId)
	if !isMemberOfNoGroups && !isViewerOfNoGroups {
		// User is probably not logged in
		return errorwithstatus.MakeBadRequestError(errors.New("User has no group membership, can't determine permissions"))
	}

	// Read in body
	imageReqData, err := io.ReadAll(params.Request.Body)
	if err != nil {
		return err
	}

	req := &protos.ImageUploadHttpRequest{}
	err = proto.Unmarshal(imageReqData, req)
	if err != nil {
		return err
	}

	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 255); err != nil {
		return err
	}

	// We only allow a few formats:
	nameLowerCase := strings.ToLower(req.Name)
	if !strings.HasSuffix(nameLowerCase, ".png") && !strings.HasSuffix(nameLowerCase, ".jpg") && !strings.HasSuffix(nameLowerCase, ".tif") {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Unexpected format: %v. Must be either PNG, JPG or 32bit float 4-channel TIF file", req.Name))
	}

	if err := wsHelpers.CheckStringField(&req.OriginScanId, "OriginScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return err
	}

	if err := wsHelpers.CheckFieldLength(req.AssociatedScanIds, "AssociatedScanIds", 0, 10); err != nil {
		return err
	}

	// Check that user has access to this scan
	_, err = wsHelpers.CheckObjectAccessForUser(false, req.OriginScanId, protos.ObjectType_OT_SCAN, userId, memberOfGroupIds, viewerOfGroupIds, params.Svcs.MongoDB)
	if err != nil {
		return err
	}

	// Read the scan to confirm it's valid, and also so we have the instrument to save for beams (if needed)
	coll := params.Svcs.MongoDB.Collection(dbCollections.ScansName)
	scanResult := coll.FindOne(context.TODO(), bson.M{"_id": req.OriginScanId}, options.FindOne())
	if scanResult.Err() != nil {
		return errorwithstatus.MakeNotFoundError(req.OriginScanId)
	}

	scan := &protos.ScanItem{}
	err = scanResult.Decode(scan)
	if err != nil {
		return fmt.Errorf("Failed to decode scan: %v. Error: %v", req.OriginScanId, err)
	}

	// Check if data size sent is 0 - treat this as an enquiry to what part we have already (resuming)
	if len(req.ImageData) <= 0 {
		resp := &protos.ImageUploadHttpPartialInfo{}

		item, ok := filePartsRecvd[req.Name]
		if ok {
			resp.BytesReceived = item.BytesSoFar
		}

		utils.SendProtoJSON(params.Writer, resp)
		return nil
	}

	// At this point we can decide what we're dealing with... if it's a multi-part send, we have to keep saving the pieces
	// but if it's a single one (or the last piece), we process and save it
	isLast, isMultipart, err := getMultipartImageRecvState(req.Name, req.PartNo, req.TotalParts, uint64(len(req.ImageData)))
	if err != nil {
		return err
	}

	// NOTE: In multi-part situations we always save the current chunk
	if isMultipart {
		params.Svcs.Log.Infof("Saving file %v chunk %v/%v", req.Name, req.PartNo, req.TotalParts)

		err = saveChunk(req.Name, req.PartNo == 0, req.ImageData)
		if err != nil {
			return err
		}
	}

	// If we're just saving chunks, stop here
	if !isLast {
		return nil
	}

	// At this point we've "finished" receiving an image and have to store it.
	// If it's not multi-part we can operate with what's in memory but if it is multi-part we'll have to
	// read the file that was saved locally (and delete it when we're done)

	// At this point check that our local file is the same size as the remote one
	localFilePath := ""
	if isMultipart {
		localFilePath, err = verifyChunksReceived(req.Name, req.ImageByteSize)
		if err != nil {
			return err
		}
	}

	// It's the last part, so here we finish everything...

	// Save image meta in collection
	purpose := protos.ScanImagePurpose_SIP_VIEWING
	if strings.HasSuffix(nameLowerCase, ".tif") {
		meta, err := gdsfilename.ParseFileName(req.Name)
		if err != nil && meta.ProdType == "MSA" || meta.ProdType == "VIS" {
			// It's only considered an RGBU image based on strict file naming!
			purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
		}
	}

	associatedScanIds := []string{req.OriginScanId}
	if len(req.AssociatedScanIds) > 0 {
		associatedScanIds = req.AssociatedScanIds
	}

	var imgWidth, imgHeight uint32
	imgFileSize := uint64(req.ImageByteSize)

	// If it's a multi-part image, we read from the file we saved
	// otherwise we just read what's in memory
	if isMultipart {
		img, err := vips.NewImageFromFile(localFilePath, nil)
		if err != nil {
			return err
		}
		imgWidth = uint32(img.Width())
		imgHeight = uint32(img.Height())
	} else {
		imgWidth, imgHeight, err = utils.ReadImageDimensions(req.Name, req.ImageData)
		if err != nil {
			return err
		}
		imgFileSize = uint64(len(req.ImageData))
	}

	// We make the names more unique this way...
	savePath := path.Join(req.OriginScanId, req.Name)
	scanImage := utils.MakeScanImage(
		savePath,
		imgFileSize,
		protos.ScanImageSource_SI_UPLOAD,
		purpose,
		associatedScanIds,
		req.OriginScanId,
		"",
		req.GetBeamImageRef(),
		"",
		"",
		imgWidth,
		imgHeight,
	)

	asPyramid := isPyramidImport(scanImage.Width, scanImage.Height, scanImage.FileSize64)

	pyramidTileSize := uint32(1024)
	pyramidQuality := uint32(90)

	if asPyramid {
		// Set the id to be the image id
		scanImage.PyramidId = scanImage.ImagePath
		// Add other pyramid related fields

		scanImage.PyramidTileFormat = "jpg"
		if pyramidQuality >= 100 {
			scanImage.PyramidTileFormat = "png"
		}

		pyramidLevels := wsHelpers.BuildImagePyramidLevels(scanImage.Width, scanImage.Height, pyramidTileSize)

		params.Svcs.Log.Infof("Importing image %v (%vx%v) as pyramid with tile size: %v, levels: %v. Pyramid ID generated: %v", req.Name, scanImage.Width, scanImage.Height, pyramidLevels.TileSize, len(pyramidLevels.Pyramid), scanImage.PyramidId)

		err = savePyramidEntry(params.Svcs.MongoDB, scanImage.PyramidId, pyramidLevels, params.Svcs.Log)
		if err != nil {
			return err
		}
	}

	generateCoords, isDuplicate, err := saveScanImage(params.Svcs.MongoDB, scan, scanImage)
	if err != nil {
		// If this isn't an "already exists" error, we delete what we've saved already. We don't want
		// to delete otherwise though, or we'd wipe out the one that already existed!
		// TODO: Probably should do this with transactions
		if !isDuplicate {
			undoScanImagePUT(params.Svcs.MongoDB, req.Name, scanImage.ImagePath, scanImage.PyramidId, localFilePath, req.ImageByteSize, params.Svcs.Log)
		}
		return err
	}

	if len(localFilePath) > 0 {
		if asPyramid {
			err = saveImageDataAsPyramid(params.Svcs.FS, params.Svcs.Config.DatasetsBucket, savePath, localFilePath, pyramidTileSize, pyramidQuality, params.Svcs.Log)
		} else {
			err = saveImageDataFromFile(params.Svcs.FS, params.Svcs.Config.DatasetsBucket, savePath, localFilePath)
		}
	} else {
		err = saveImageDataFromMemory(params.Svcs.FS, params.Svcs.Config.DatasetsBucket, savePath, req.ImageData)
	}

	if err != nil {
		undoScanImagePUT(params.Svcs.MongoDB, req.Name, scanImage.ImagePath, scanImage.PyramidId, localFilePath, req.ImageByteSize, params.Svcs.Log)
		return err
	}

	// From this point on, we consider the image created. If IJ generation fails or anything else, we don't roll back and delete

	if generateCoords {
		_, err = wsHelpers.GenerateIJs(scanImage.ImagePath, req.OriginScanId, scan.Instrument, params.Svcs)
		if err != nil {
			return err
		}
	}

	// Finally, update the scan if needed
	err = wsHelpers.UpdateScanImageDataTypes(req.OriginScanId, params.Svcs.MongoDB, params.Svcs.Log)
	if err != nil {
		params.Svcs.Log.Errorf("UpdateScanImageDataTypes Failed for scan: %v, when uploading image: %v. DataType counts may not be accurate on Scan Item, RGBU icon may not show correctly.", req.OriginScanId, scanImage.ImagePath)
	}

	// Notify of our successful image addition
	params.Svcs.Notifier.NotifyNewScanImage(req.OriginScanId, req.OriginScanId, scanImage.ImagePath)
	params.Svcs.Notifier.SysNotifyScanImagesChanged(scanImage.ImagePath, scanImage.AssociatedScanIds)

	return nil
}

func savePyramidEntry(db *mongo.Database, pyramidId string, pyramidInfo *protos.ImagePyramid, l logger.ILogger) error {
	entry := &protos.ImagePyramidDBEntry{
		Id:      pyramidId,
		Pyramid: pyramidInfo,
	}

	coll := db.Collection(dbCollections.ImagePyramidsName)
	opt := options.InsertOne()
	result, err := coll.InsertOne(context.TODO(), entry, opt)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// Don't overwrite, so we're OK with this
			//return nil
			l.Errorf("insertImagePyramid writing pyramid structure to DB skipped due to duplicate pyramid id: %v", pyramidId)
		} else {
			// A real error happened!
			return err
		}
	} else if result.InsertedID != entry.Id {
		l.Errorf("insertImagePyramid wrote id %v, got back %v", entry.Id, result.InsertedID)
		// Not the end of the world... don't error out here
	}

	return nil
}

// Returns generate flag, is duplicate flag, and error
func saveScanImage(db *mongo.Database, scan *protos.ScanItem, scanImage *protos.ScanImage) (bool, bool, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagesName)

	// If this is the first image added to a dataset that has no images (and hence no beam location ij's), generate ij's here so the image can be
	// aligned to them. The image will refer to itself as the owner of the ij's it's matching and will be able to have a transform too
	foundItems, err := coll.Find(ctx, bson.M{"originscanid": scanImage.OriginScanId}, options.Find())
	// If there was an error, stop here
	if err != nil && err != mongo.ErrNoDocuments {
		return false, false, fmt.Errorf("Error while querying for other images for scan %v. Error was: %v", scanImage.OriginScanId, err)
	}

	generateCoords := err == mongo.ErrNoDocuments // This won't really happen... Find() doesn't return an error for none!
	if !generateCoords {
		// Check if the count is 0
		generateCoords = !foundItems.Next(ctx)
	}

	hasXRF := false
	for _, dt := range scan.DataTypes {
		if dt.DataType == protos.ScanDataType_SD_XRF && dt.Count > 0 {
			hasXRF = true
			break
		}
	}

	if !hasXRF {
		generateCoords = false
	}

	if generateCoords && scanImage.MatchInfo == nil {
		// Set a beam transform
		scanImage.MatchInfo = &protos.ImageMatchTransform{
			BeamImageFileName: scanImage.ImagePath,
			XOffset:           0,
			YOffset:           0,
			XScale:            1,
			YScale:            1,
		}
	}

	result, err := coll.InsertOne(ctx, scanImage, options.InsertOne())
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return false, true, errorwithstatus.MakeBadRequestError(fmt.Errorf("%v already exists", scanImage.ImagePath))
		}
		return generateCoords, false, err
	}

	if result.InsertedID != scanImage.ImagePath {
		return generateCoords, false, fmt.Errorf("HandleImageUploadReq wrote id %v, got back %v", scanImage.ImagePath, result.InsertedID)
	}

	return generateCoords, false, nil
}

// Call to abort an image upload operation - if anything happens here we delete all the things that the
// image upload may have affected/saved/created
func undoScanImagePUT(db *mongo.Database, reqName string, scanImageId string, pyramidId string, localFilePath string, imageByteSize uint64, l logger.ILogger) {
	// DB Image
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagesName)
	filter := bson.D{{Key: "_id", Value: scanImageId}}
	delOpt := options.Delete()
	_ /*delImgResult*/, err := coll.DeleteOne(ctx, filter, delOpt)
	if err != nil {
		l.Errorf("Failed to delete scan image %v while handling image upload error. Error was: %v", scanImageId, err)
	} else {
		l.Infof("Deleted imported scan image %v due to image upload error", scanImageId)
	}

	// DB ImagePyramid
	coll = db.Collection(dbCollections.ImagePyramidsName)
	filter = bson.D{{Key: "_id", Value: pyramidId}}
	delOpt = options.Delete()
	_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
	if err != nil {
		l.Errorf("Failed to delete scan image pyramid %v while handling image upload error. Error was: %v", pyramidId, err)
	} else {
		l.Infof("Deleted imported scan image pyramid %v due to image upload error", pyramidId)
	}

	// Upload parts received log
	delete(filePartsRecvd, reqName)

	// Local image file (containing uploaded chunks)
	if err2 := os.Remove(localFilePath); err2 != nil {
		l.Errorf("Failed to delete local copy of downloaded image %v, error: %v", localFilePath, err2)
	} else {
		l.Infof("Deleted temp download image %v, size: %v", localFilePath, imageByteSize)
	}
}

func saveImageDataAsPyramid(
	fs fileaccess.FileAccess,
	bucket string,
	savePath string,
	localImageDataPath string,
	tileSize uint32,
	tileQuality uint32,
	l logger.ILogger) error {
	// This is the most complex situation. We take the locally saved file (constructed from chunks sent) and
	// break it up into tiles. We write these to S3 as separate images in a tree structure

	// Read the image
	img, err := vips.NewImageFromFile(localImageDataPath, &vips.LoadOptions{ /*Page: , N: */ })
	if err != nil {
		return err
	}

	// Use vipsgen Dzsave
	// IMPORTANT: These options are carefully chosen:
	// - Imagename: The base name for generated files
	// - Suffix: .jpg for JPEG tiles
	// - Q: JPEG quality (85 = good balance of quality/size)
	// - Depth: DzDepthOnetile means generate tiles at all zoom levels
	// - Overlap: 0 means no pixel overlap between tiles (can be changed if needed)
	// - TileSize: Size of each tile (254 is default, accounts for overlap)
	// If quality is 100% we do PNG output
	suffix := ".jpg"
	if tileQuality == 100 {
		suffix = ".png"
	}

	l.Infof("  Generating tiles of size %v, quality: %v, format: %v...", tileSize, tileQuality, suffix)

	err = img.Dzsave(localImageDataPath, &vips.DzsaveOptions{
		Imagename: filepath.Base(localImageDataPath),
		Suffix:    suffix,
		Q:         int(tileQuality),
		Depth:     vips.DzDepthOnetile,
		Overlap:   0,
		TileSize:  int(tileSize),
	})

	if err != nil {
		return err
	}

	// Write to S3 (need to use a multithreaded approach as it'd take way too long otherwise)
	// Save the image we saved to local storage in chunks as one file to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	return fileaccess.CopyToBucket(fs, "", localImageDataPath+"_files", bucket, s3Path, true, l)
}

func saveImageDataFromFile(fs fileaccess.FileAccess, bucket string, savePath string, localImageDataPath string) error {
	// Save the image we saved to local storage in chunks as one file to S3
	s3Path := filepaths.GetImageFilePath(savePath)

	f, err := os.Open(localImageDataPath)
	if err != nil {
		return err
	}
	return fs.WriteObjectStream(bucket, s3Path, f)
}

func saveImageDataFromMemory(fs fileaccess.FileAccess, bucket string, savePath string, imageData []byte) error {
	// Save the image we have in memory to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	return fs.WriteObject(bucket, s3Path, imageData)
}

func isPyramidImport(imageWidth uint32, imageHeight uint32, imageFileSize uint64) bool {
	return imageWidth > 4000 || imageHeight > 4000 || imageFileSize > 35*1024*1024
}

func getImageChunkPath(fileName string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Join(wd, "image-cache", fileName), nil
}

// Returns: isLast, isMultiPart, error
// Where: Error is nil unless something doesn't make sense
//
//	isLast is true if this is the "last" part of the file we're dealing with
//	isMultiPart is true if the image is being received in multiple parts
func getMultipartImageRecvState(fileName string, partNo uint32, totalParts uint32, byteLength uint64) (bool, bool, error) {
	// If it's a single part download, just stop here, it's the "final" part already
	// NOTE: we're treating total=0 the same as 1, I guess in case something sends
	// us an image without having updated the code, the field will be 0 and we can
	// treat that just like we treat a 1 anyway...
	if totalParts <= 1 {
		return true, false, nil
	}

	// It's more than 1 part, if we've got parts for it before we can verify a few things...
	if item, ok := filePartsRecvd[fileName]; !ok {
		// We don't have a log item for this, save one
		filePartsRecvd[fileName] = FilePartRecvItem{
			LastPartNo: partNo,
			TotalParts: totalParts,
			BytesSoFar: byteLength,
		}

		// Expecting more, signal to save the chunk
		return false, true, nil
	} else {
		// We have a record already, check that we've got the next sequential part down
		if partNo != item.LastPartNo+1 {
			return false, true, fmt.Errorf("Expected file part number: %v, got: %v", item.LastPartNo+1, partNo)
		}

		if partNo >= totalParts {
			return false, true, fmt.Errorf("Unexpected file part number: %v for total %v", partNo, totalParts)
		}

		if totalParts != item.TotalParts {
			return false, true, fmt.Errorf("Total parts changed from: %v, to: %v", item.TotalParts, totalParts)
		}

		// Update and save
		item.BytesSoFar += byteLength
		item.LastPartNo = partNo

		filePartsRecvd[fileName] = item

		// If it's the last part, process it as such
		return item.LastPartNo == item.TotalParts-1, true, nil
	}
}

func saveChunk(fileName string, truncate bool, data []byte) error {
	// Write it
	imgPath, err := getImageChunkPath(fileName)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(imgPath), 0777)
	if err != nil {
		return err
	}

	flags := os.O_CREATE | os.O_WRONLY
	if truncate {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}

	f, err := os.OpenFile(imgPath, flags, 0777)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	err = f.Sync()
	if err != nil {
		return err
	}

	return f.Close()
}

// Returns local image path, error if needed
func verifyChunksReceived(fileName string, expectedSize uint64) (string, error) {
	imgPath, err := getImageChunkPath(fileName)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(imgPath)
	if err != nil {
		return imgPath, err
	}

	if info.Size() != int64(expectedSize) {
		return imgPath, fmt.Errorf("Unexpected upload size %v for file \"%v\", expected %v bytes", info.Size(), fileName, expectedSize)
	}

	// TODO: Checksum stuff?
	return imgPath, nil
}

func checkImpersonation(svcs *services.APIServices, userId string) (string, error) {
	if !svcs.Config.ImpersonateEnabled {
		return userId, nil
	}

	coll := svcs.MongoDB.Collection(dbCollections.UserImpersonatorsName)
	ctx := context.TODO()
	impersonateResult := coll.FindOne(ctx, bson.M{"_id": userId}, options.FindOne())
	if impersonateResult.Err() != nil {
		if impersonateResult.Err() != mongo.ErrNoDocuments {
			return userId, impersonateResult.Err()
		}
		return userId, nil
	}

	// We got impersonation info, find the user id we want to pretend to be
	item := wsHelpers.UserImpersonationItem{}
	err := impersonateResult.Decode(&item)

	if err != nil {
		return userId, err
	}

	return item.ImpersonatedId, nil
}
