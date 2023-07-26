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
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/imageedit"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
	requestedFileName := params.PathParams[FileNameIdentifier]

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, ok := wsHelpers.GetCachedUserGroupMembership(params.UserInfo.UserID)
	if !ok {
		// User is probably not logged in
		return nil, "", "", "", 0, errorwithstatus.MakeBadRequestError(errors.New("User group membership not found, can't determine permissions"))
	}

	// Now read the DB record for the image, so we can determine what scans it's associated with
	ctx := context.TODO()
	coll := params.Svcs.MongoDB.Collection(dbCollections.ImagesName)

	filter := bson.M{"_id": requestedFileName}
	imageDBResult := coll.FindOne(ctx, filter)

	if imageDBResult.Err() != nil {
		// This doesn't look good...
		if imageDBResult.Err() == mongo.ErrNoDocuments {
			return nil, "", "", "", 0, errorwithstatus.MakeNotFoundError(requestedFileName)
		}
		return nil, "", "", "", 0, imageDBResult.Err()
	}

	dbImage := protos.ScanImage{}
	err := imageDBResult.Decode(&dbImage)
	if err != nil {
		return nil, "", "", "", 0, err
	}

	for _, scanId := range dbImage.AssociatedScanIds {
		_, err := wsHelpers.CheckObjectAccessForUser(false, scanId, protos.ObjectType_OT_SCAN, params.UserInfo.UserID, memberOfGroupIds, params.Svcs.MongoDB)
		if err != nil {
			return nil, "", "", "", 0, err
		}
	}

	// We're still here, so we have access! Check query params for any modifiers
	finalFileName := requestedFileName
	showLocations, err := getBoolValue(params.PathParams["with-locations"])
	if err != nil {
		return nil, "", "", "", 0, err
	} else if showLocations {
		finalFileName = addFileNameSuffix(finalFileName, "-withloc")
	}

	isThumb, err := getBoolValue(params.PathParams["thumbnail"])
	if err != nil {
		return nil, "", "", "", 0, err
	} else if isThumb {
		finalFileName = addFileNameSuffix(finalFileName, "-thumbnail")
	}

	statuscode := 200

	// Load from dataset directory unless custom loading is requested, where we look up the file in the manual bucket
	imgBucket := params.Svcs.Config.DatasetsBucket

	// Check if the file exists, as we may be able to generate a version of the underlying file to satisfy the request
	var s3Path string
	if isThumb || showLocations {
		s3Path = filepaths.GetImageCacheFilePath(path.Join(scanID, finalFileName))
	} else {
		s3Path = filepaths.GetImageFilePath(path.Join(scanID, finalFileName))
	}

	_, err = params.Svcs.S3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(imgBucket),
		Key:    aws.String(s3Path),
	})
	if err != nil {
		if isThumb || showLocations {
			// If the file doesn't exist, check if the base file name exists, because we may just need to generate
			// a modified version of it
			genS3Path := filepaths.GetImageFilePath(path.Join(scanID, requestedFileName))

			// Original file exists, generate this modified copy and cache it back in S3 for the rest of this
			// function to find!
			err = generateImageVersion(genS3Path, isThumb, showLocations, s3Path, params.Svcs)
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

func generateImageVersion(s3Path string, thumbnail bool, showLocations bool, finalFilePath string, svcs *services.APIServices) error {
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

		filter := bson.M{"_id": path.Base(s3Path)}
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

	if thumbnail {
		// We want it to be a max of 200px across
		img = imageedit.ScaleImage(img, 200)
	}

	// Now we're done, read the bytes out in the right format
	imgBytesOut, err := imageedit.GetImageBytes(img, imgFormat)
	if err != nil {
		return err
	}

	// Write the final image to final cached destination
	return svcs.FS.WriteObject(svcs.Config.DatasetsBucket, finalFilePath, imgBytesOut)
}
