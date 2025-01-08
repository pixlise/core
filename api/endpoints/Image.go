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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/imageedit"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, isMemberOfNoGroups := wsHelpers.GetCachedUserGroupMembership(params.UserInfo.UserID)
	viewerOfGroupIds, isViewerOfNoGroups := wsHelpers.GetCachedUserGroupViewership(params.UserInfo.UserID)
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
		_, err := wsHelpers.CheckObjectAccessForUser(false, scanId, protos.ObjectType_OT_SCAN, params.UserInfo.UserID, memberOfGroupIds, viewerOfGroupIds, params.Svcs.MongoDB)
		if err != nil {
			return nil, "", "", "", 0, err
		}
	}

	// We're still here, so we have access! Check query params for any modifiers
	finalFileName := dbImage.ImagePath
	showLocations, err := getBoolValue(params.PathParams["with-locations"])
	if err != nil {
		return nil, "", "", "", 0, err
	} else if showLocations {
		finalFileName = addFileNameSuffix(finalFileName, "-withloc")
	}

	var minWidthPx = 0
	if minWStr, ok := params.PathParams["minwidth"]; ok {
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
			genS3Path := filepaths.GetImageFilePath(requestedFileName)

			// Original file exists, generate this modified copy and cache it back in S3 for the rest of this
			// function to find!
			err = generateImageVersion(requestedFileName, genS3Path, minWidthPx, showLocations, s3Path, params.Svcs)
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

	return result, requestedFileName, etag, lm.String(), 0, err
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

		filter := bson.M{"_id": wsHelpers.GetImageNameSansVersion(imageName)}
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

func PutImage(params apiRouter.ApiHandlerGenericParams) error {
	if !params.UserInfo.Permissions["EDIT_SCAN"] {
		return errorwithstatus.MakeBadRequestError(errors.New("PutImage not allowed"))
	}

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, isMemberOfNoGroups := wsHelpers.GetCachedUserGroupMembership(params.UserInfo.UserID)
	viewerOfGroupIds, isViewerOfNoGroups := wsHelpers.GetCachedUserGroupViewership(params.UserInfo.UserID)
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
	_, err = wsHelpers.CheckObjectAccessForUser(false, req.OriginScanId, protos.ObjectType_OT_SCAN, params.UserInfo.UserID, memberOfGroupIds, viewerOfGroupIds, params.Svcs.MongoDB)
	if err != nil {
		return err
	}

	// Read the scan to confirm it's valid, and also so we have the instrument to save for beams (if needed)
	ctx := context.TODO()
	coll := params.Svcs.MongoDB.Collection(dbCollections.ScansName)
	scanResult := coll.FindOne(ctx, bson.M{"_id": req.OriginScanId}, options.FindOne())
	if scanResult.Err() != nil {
		return errorwithstatus.MakeNotFoundError(req.OriginScanId)
	}

	scan := &protos.ScanItem{}
	err = scanResult.Decode(scan)
	if err != nil {
		return fmt.Errorf("Failed to decode scan: %v. Error: %v", req.OriginScanId, err)
	}

	db := params.Svcs.MongoDB

	// Save image meta in collection
	imgWidth, imgHeight, err := utils.ReadImageDimensions(req.Name, req.ImageData)
	if err != nil {
		return err
	}

	purpose := protos.ScanImagePurpose_SIP_VIEWING
	if strings.HasSuffix(nameLowerCase, ".tif") {
		purpose = protos.ScanImagePurpose_SIP_MULTICHANNEL
	}

	associatedScanIds := []string{req.OriginScanId}
	if len(req.AssociatedScanIds) > 0 {
		associatedScanIds = req.AssociatedScanIds
	}

	// We make the names more unique this way...
	savePath := path.Join(req.OriginScanId, req.Name)
	scanImage := utils.MakeScanImage(
		savePath,
		uint32(len(req.ImageData)),
		protos.ScanImageSource_SI_UPLOAD,
		purpose,
		associatedScanIds,
		req.OriginScanId,
		"",
		req.GetBeamImageRef(),
		imgWidth,
		imgHeight,
	)

	coll = db.Collection(dbCollections.ImagesName)

	// If this is the first image added to a dataset that has no images (and hence no beam location ij's), generate ij's here so the image can be
	// aligned to them. The image will refer to itself as the owner of the ij's it's matching and will be able to have a transform too
	foundItems, err := coll.Find(ctx, bson.M{"originscanid": req.OriginScanId}, options.Find())
	// If there was an error, stop here
	if err != nil && err != mongo.ErrNoDocuments {
		return fmt.Errorf("Error while querying for other images for scan %v. Error was: %v", req.OriginScanId, err)
	}

	generateCoords := err == mongo.ErrNoDocuments // This won't really happen... Find() doesn't return an error for none!
	if !generateCoords {
		// Check if the count is 0
		generateCoords = !foundItems.Next(ctx)
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
			return errorwithstatus.MakeBadRequestError(fmt.Errorf("%v already exists", scanImage.ImagePath))
		}
		return err
	}

	if result.InsertedID != scanImage.ImagePath {
		return fmt.Errorf("HandleImageUploadReq wrote id %v, got back %v", scanImage.ImagePath, result.InsertedID)
	}

	// Save the image to S3
	s3Path := filepaths.GetImageFilePath(savePath)
	err = params.Svcs.FS.WriteObject(params.Svcs.Config.DatasetsBucket, s3Path, req.ImageData)
	if err != nil {
		// Failed to upload image data, so no point in having a DB entry now either...
		coll = params.Svcs.MongoDB.Collection(dbCollections.ImagesName)
		filter := bson.D{{Key: "_id", Value: scanImage.ImagePath}}
		delOpt := options.Delete()
		_ /*delImgResult*/, err = coll.DeleteOne(ctx, filter, delOpt)
		return err
	}

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
