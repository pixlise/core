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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/awsutil"
	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/utils"
	dataConverter "github.com/pixlise/core/v2/data-import/data-converter"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/services"
)

// NOTE: No registration function here, this sits in the dataset registration function, shares paths with it. Only separated
// into this file for code clarity

const customImageTypeUnaligned = "unaligned"
const customImageTypeMatched = "matched"
const customImageTypeRGBU = "rgbu"

func isValidCustomImageType(imgType string) bool {
	return (imgType == customImageTypeUnaligned ||
		imgType == customImageTypeMatched ||
		imgType == customImageTypeRGBU)
}

////////////////////////////////////////////////////////////////////////
// Dataset regenerate request

type datasetReprocessSNSRequest struct {
	DatasetID string `json:"datasetID"`
	LogID     string `json:"logID"`
}

type datasetReprocessResponse struct {
	LogID string
}

func datasetReprocess(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	result, logId, err := triggerDatasetReprocessViaSNS(params.Svcs.SNS, params.Svcs.IDGen, datasetID, params.Svcs.Config.DataSourceSNSTopic)

	params.Svcs.Log.Infof("Published SNS topic: %v. Log ID: %v", result, logId)
	return nil, err
}

func triggerDatasetReprocessViaSNS(snsSvc awsutil.SNSInterface, idGen services.IDGenerator, datasetID string, snsTopic string) (*sns.PublishOutput, string, error) {
	// Generate a new log ID that this reprocess job will write to
	// which we also return to the caller, so they can track what happens
	// with this async task

	reprocessId := fmt.Sprintf("dataimport-%s", idGen.GenObjectID())

	snsReq := datasetReprocessSNSRequest{
		DatasetID: datasetID,
		LogID:     reprocessId,
	}

	snsReqJSON, err := json.Marshal(snsReq)
	if err != nil {
		return nil, "", api.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to trigger dataset reprocess: %v", err))
	}

	result, err := snsSvc.Publish(&sns.PublishInput{
		Message:  aws.String(string(snsReqJSON)),
		TopicArn: aws.String(snsTopic),
	})

	if err != nil {
		return nil, "", api.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to publish SNS topic for dataset regeneration: %v", err))
	}

	return result, reprocessId, nil
}

////////////////////////////////////////////////////////////////////////
// Meta data get/set

func datasetCustomMetaGet(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	s3Path := filepaths.GetCustomMetaPath(datasetID)

	meta := dataConverter.DatasetCustomMeta{}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.ManualUploadBucket, s3Path, &meta, false)

	if err != nil {
		return nil, api.MakeNotFoundError("dataset custom meta")
	}

	return meta, nil
}

func datasetCustomMetaPut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	s3Path := filepaths.GetCustomMetaPath(datasetID)

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req dataConverter.DatasetCustomMeta
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Create/overwrite what's there
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ManualUploadBucket, s3Path, req)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

////////////////////////////////////////////////////////////////////////
// Custom image list/get/add/delete

type datasetCustomImageMeta struct {
	DownloadLink string `json:"download-link"`
}
type datasetCustomMatchedImageMeta struct {
	AlignedImageLink string `json:"alignedImageLink"`
	*datasetCustomImageMeta
	*datasetModel.MatchedAlignedImageMeta
}

func datasetCustomImagesList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	imgType := params.PathParams[customImageTypeIdentifier]

	if !isValidCustomImageType(imgType) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid custom image type: \"%v\"", imgType))
	}

	// customImageTypeUnaligned, customImageTypeMatched, customImageTypeRGBU
	s3Path := filepaths.GetCustomImagePath(datasetID, imgType, "")

	// List all files in the path - this way we get all view states back and know the types to read as it's encoded in the file names
	items, err := params.Svcs.FS.ListObjects(params.Svcs.Config.ManualUploadBucket, s3Path+"/")
	if err != nil {
		return nil, api.MakeNotFoundError("custom images")
	}

	// Return just the file names
	fileNames := []string{}
	for _, item := range items {
		fileNames = append(fileNames, path.Base(item))
	}

	// If we're reading matched images, filter out JSON files. Uploader forces the JSON and image to have the same name (just differs by ext)
	if imgType == customImageTypeMatched {
		// JSON and image files should have the same name, so we only return the image file name corresponding to such a pair
		filesAndExtensions := map[string][]string{}

		for _, item := range fileNames {
			dotPos := strings.LastIndex(item, ".")
			if dotPos > 0 {
				filePart := item[0:dotPos]
				filesAndExtensions[filePart] = append(filesAndExtensions[filePart], item[dotPos:])
			}
		}

		// Now run through and return the not-json one of each
		fileNames = []string{}
		for filePart, extensions := range filesAndExtensions {
			notJSONext := ""
			foundJSON := false
			for _, e := range extensions {
				if e == ".json" {
					foundJSON = true
				} else {
					notJSONext = e
				}
			}

			if foundJSON && len(notJSONext) > 0 {
				fileNames = append(fileNames, filePart+notJSONext)
			}
		}

		// Sort the file names returned, just because they come out of a map and Go map order is non-deterministic, so sorting
		// this will ensure repeatable unit tests
		sort.Strings(fileNames)
	}

	// Return the combined set
	return &fileNames, nil
}

func datasetCustomImageGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// Here we don't actually return the image, that's done through /download. We return any metadata
	// For most images this will simply end up containing the /download link. For MATCHED images, we
	// return the contents of the JSON file too.
	datasetID := params.PathParams[datasetIdentifier]
	imgType := params.PathParams[customImageTypeIdentifier]

	if !isValidCustomImageType(imgType) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid custom image type: \"%v\"", imgType))
	}

	fileName := params.PathParams[customImageIdentifier]

	downloadLink := params.PathParams[handlers.HostParamName] + "/" + path.Join(datasetPathPrefix, handlers.UrlStreamDownloadIndicator, datasetID, fileName) + "?loadCustomType=" + imgType
	result := datasetCustomImageMeta{
		DownloadLink: downloadLink,
	}

	// For matched images, we need to read the associated JSON file. User should be requesting the image file name
	// but this matches the JSON file except by extension...
	if imgType == customImageTypeMatched {
		dotPos := strings.LastIndex(fileName, ".")
		if dotPos < 0 {
			return nil, api.MakeBadRequestError(fmt.Errorf("Invalid file name: \"%v\"", fileName))
		}

		// Form the JSON file name
		fileName = fileName[0:dotPos] + ".json"

		// Get the JSON file
		s3Path := filepaths.GetCustomImagePath(datasetID, imgType, fileName)

		var imgMeta datasetModel.MatchedAlignedImageMeta
		err := params.Svcs.FS.ReadJSON(params.Svcs.Config.ManualUploadBucket, s3Path, &imgMeta, false)

		if err != nil {
			return nil, api.MakeNotFoundError("dataset custom image meta")
		}

		// Find a link to the image this one is aligned to if possible
		alignedImageLink := ""

		datasetPath := filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetFileName)
		dataset, err := datasetModel.GetDataset(params.Svcs, datasetPath)

		if err == nil {
			// Find the aligned image for this PMC in the dataset
			for _, img := range dataset.AlignedContextImages {
				if imgMeta.AlignedBeamPMC == img.Pmc {
					alignedImageLink = params.PathParams[handlers.HostParamName] + "/" + path.Join(datasetPathPrefix, handlers.UrlStreamDownloadIndicator, datasetID, img.Image)
					break
				}
			}
		}

		matchedResult := datasetCustomMatchedImageMeta{
			alignedImageLink,
			&result,
			&imgMeta,
		}

		return &matchedResult, nil
	}

	// Just return the download link
	return &result, nil
}

func datasetCustomImagesPost(params handlers.ApiHandlerParams) (interface{}, error) {
	// Expecting post body to be the image contents, file name is in the path.
	// If uploading a matched image type, we expect query parameters:
	// x-offset, y-offset, x-scale, y-scale, aligned-image
	datasetID := params.PathParams[datasetIdentifier]
	imgType := params.PathParams[customImageTypeIdentifier]

	if !isValidCustomImageType(imgType) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid custom image type: \"%v\"", imgType))
	}

	fileName := params.PathParams[customImageIdentifier]
	dotPos := strings.LastIndex(fileName, ".")
	if dotPos < 0 {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid file name: \"%v\"", fileName))
	}

	// Check file extension is valid
	allowedExt := []string{".png", ".jpg"}
	if imgType == customImageTypeRGBU {
		allowedExt = []string{".tif"}
	} else if imgType == customImageTypeMatched {
		allowedExt = append(allowedExt, ".tif")
	}

	if !utils.StringInSlice(fileName[dotPos:], allowedExt) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid image file type: \"%v\"", fileName))
	}

	// Write the image file to S3
	imgData, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	if len(imgData) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("No image data sent"))
	}

	// If we're receiving a matched image type, we expect the extra parameters...
	if imgType == customImageTypeMatched {
		floatValues, alignedPMC, err := parseMatchedImageFromParams(params)
		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}

		// Form a JSON we can upload for this
		meta := datasetModel.MatchedAlignedImageMeta{
			XOffset:          floatValues[0],
			YOffset:          floatValues[1],
			XScale:           floatValues[2],
			YScale:           floatValues[3],
			AlignedBeamPMC:   int32(alignedPMC),
			MatchedImageName: fileName,
		}

		s3Path := filepaths.GetCustomImagePath(datasetID, imgType, fileName[0:dotPos]+".json")
		err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ManualUploadBucket, s3Path, meta)
		if err != nil {
			return nil, err
		}
	}

	s3Path := filepaths.GetCustomImagePath(datasetID, imgType, fileName)
	err = params.Svcs.FS.WriteObject(params.Svcs.Config.ManualUploadBucket, s3Path, imgData)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Allows editing of matched image parameters
func datasetCustomImagesPut(params handlers.ApiHandlerParams) (interface{}, error) {
	// Expecting post body to be the image contents, file name is in the path.
	// If uploading a matched image type, we expect query parameters:
	// x-offset, y-offset, x-scale, y-scale, aligned-image
	datasetID := params.PathParams[datasetIdentifier]
	imgType := params.PathParams[customImageTypeIdentifier]

	if imgType != customImageTypeMatched {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid custom image type: \"%v\"", imgType))
	}

	// Read the new values to set
	floatValues, alignedPMC, err := parseMatchedImageFromParams(params)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// See if the file exists
	fileName := params.PathParams[customImageIdentifier]
	dotPos := strings.LastIndex(fileName, ".")
	if dotPos < 0 {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid file name: \"%v\"", fileName))
	}

	metaFileName := fileName[0:dotPos] + ".json"

	// Get the JSON file
	s3Path := filepaths.GetCustomImagePath(datasetID, imgType, metaFileName)

	var existingMeta datasetModel.MatchedAlignedImageMeta
	err = params.Svcs.FS.ReadJSON(params.Svcs.Config.ManualUploadBucket, s3Path, &existingMeta, false)

	if err != nil {
		return nil, api.MakeNotFoundError(metaFileName)
	}

	// Form a JSON we can upload for this
	metaToSave := datasetModel.MatchedAlignedImageMeta{
		XOffset:          floatValues[0],
		YOffset:          floatValues[1],
		XScale:           floatValues[2],
		YScale:           floatValues[3],
		AlignedBeamPMC:   int32(alignedPMC),
		MatchedImageName: fileName,
	}

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ManualUploadBucket, s3Path, metaToSave)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func parseMatchedImageFromParams(params handlers.ApiHandlerParams) ([]float32, int32, error) {
	fileName := params.PathParams[customImageIdentifier]

	expectedFields := []string{"x-offset", "y-offset", "x-scale", "y-scale", "aligned-beam-pmc"}
	expectFloat := []bool{true, true, true, true, false}
	floatValues := []float32{}
	alignedPMC := -1

	for c, f := range expectedFields {
		if v, ok := params.PathParams[f]; !ok {
			return floatValues, int32(alignedPMC), fmt.Errorf("Missing query parameter \"%v\" for matched image: \"%v\"", f, fileName)
		} else {
			if expectFloat[c] {
				fVal, err := strconv.ParseFloat(v, 32)
				if err != nil {
					return floatValues, int32(alignedPMC), fmt.Errorf("Query parameter \"%v\" was not a float, for matched image: \"%v\"", f, fileName)
				}

				floatValues = append(floatValues, float32(fVal))
			} else {
				iVal, err := strconv.ParseInt(v, 10, 32)
				if err != nil {
					return floatValues, int32(alignedPMC), fmt.Errorf("Query parameter \"%v\" was not an int, for matched image: \"%v\"", f, fileName)
				}

				alignedPMC = int(iVal)
			}
		}
	}

	if alignedPMC < 0 {
		return floatValues, int32(alignedPMC), fmt.Errorf("No or invalid aligned beam PMC specified, when for image: \"%v\"", fileName)
	}

	return floatValues, int32(alignedPMC), nil
}

func datasetCustomImagesDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	imgType := params.PathParams[customImageTypeIdentifier]

	if !isValidCustomImageType(imgType) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid custom image type: \"%v\"", imgType))
	}

	fileName := params.PathParams[customImageIdentifier]

	// Matched images: delete the corresponding JSON file too
	if imgType == customImageTypeMatched {
		dotPos := strings.LastIndex(fileName, ".")
		if dotPos < 0 {
			return nil, api.MakeBadRequestError(fmt.Errorf("Invalid file name: \"%v\"", fileName))
		}

		// Form the JSON file name
		jsonFileName := fileName[0:dotPos] + ".json"

		// Get the JSON file
		s3Path := filepaths.GetCustomImagePath(datasetID, imgType, jsonFileName)
		err := params.Svcs.FS.DeleteObject(params.Svcs.Config.ManualUploadBucket, s3Path)
		if err != nil {
			return nil, api.MakeNotFoundError(jsonFileName)
		}
	}

	// Delete the image
	s3Path := filepaths.GetCustomImagePath(datasetID, imgType, fileName)

	err := params.Svcs.FS.DeleteObject(params.Svcs.Config.ManualUploadBucket, s3Path)
	if err != nil {
		return nil, api.MakeNotFoundError(fileName)
	}
	return nil, nil
}
