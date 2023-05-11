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
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/api/filepaths"
	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/fileaccess"
	datasetArchive "github.com/pixlise/core/v3/data-import/dataset-archive"
	"github.com/pixlise/core/v3/data-import/importer"
)

const datasetURLEnd = "dataset"
const diffractionURLEnd = "diffraction"
const contextImageReqName = "context-image"
const contextThumbnailReqName = "context-thumb"
const contextMarkupThumbnailReqName = "context-mark-thumb"

// We add links to the summary...
// TODO: is this even required any more? We've standardised on dataset filenames, etc... Maybe context image is still needed?

const datasetPathPrefix = "dataset"
const customImageTypeIdentifier = "imgtype"
const customImageIdentifier = "image"

var allowedQueryNames = map[string]bool{
	"dataset_id":      true,
	"group_id":        true,
	"sol":             false, // can be > or <
	"rtt":             false, // can be > or <
	"sclk":            false, // can be > or <
	"data_file_size":  false, // can be > or <
	"location_count":  false, // can be > or <
	"dwell_spectra":   false, // can be > or <
	"normal_spectra":  false, // can be > or <
	"target_id":       true,
	"site_id":         true,
	"drive_id":        true,
	"detector_config": true,
	"title":           true,
}

func registerDatasetHandler(router *apiRouter.ApiObjectRouter) {
	// Listing datasets (tiles screen)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermPublic), datasetListing)

	// Creating datasets
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataset), datasetCreatePost)

	// Regeneration/manual editing of datasets
	// Setting/getting meta fields
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/meta", datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), datasetCustomMetaGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/meta", datasetIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataset), datasetCustomMetaPut)

	// Reprocess
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/reprocess", datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermReadDataAnalysis), datasetReprocess)

	// Adding/viewing/removing extra images (eg WATSON)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/images", datasetIdentifier, customImageTypeIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), datasetCustomImagesList)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/images", datasetIdentifier, customImageTypeIdentifier, customImageIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), datasetCustomImageGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/images", datasetIdentifier, customImageTypeIdentifier, customImageIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataset), datasetCustomImagesPost)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/images", datasetIdentifier, customImageTypeIdentifier, customImageIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataset), datasetCustomImagesPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/images", datasetIdentifier, customImageTypeIdentifier, customImageIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataset), datasetCustomImagesDelete)

	// Streaming from S3
	router.AddCacheControlledStreamHandler(handlers.MakeEndpointPath(datasetPathPrefix+"/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), datasetFileStream)
}

func readDataSetData(svcs *services.APIServices, s3Path string) (datasetModel.DatasetConfig, error) {
	allDatasets := datasetModel.DatasetConfig{}
	return allDatasets, svcs.FS.ReadJSON(svcs.Config.ConfigBucket, s3Path, &allDatasets, false)
}

type queryItem struct {
	name     string
	operator string
	value    string
}

func (q queryItem) compareI(value int) (bool, error) {
	iQueryValue, err := strconv.Atoi(q.value)
	if err != nil {
		return false, fmt.Errorf("Failed to compare, value %v was not integer", q.value)
	}

	if q.operator == ">" {
		return value > iQueryValue, nil
	} else if q.operator == "<" {
		return value < iQueryValue, nil
	}
	//else if q.operator == "=" {
	return iQueryValue == value, nil
	//}
}

func (q queryItem) compareS(value string) (bool, error) {
	if q.operator != "=" {
		return false, fmt.Errorf("Failed to compare %v, can only use = for values \"%v\", \"%v\"", q.name, q.value, value)
	}

	return q.value == value, nil
}

func (q queryItem) compareSAllowIntConvert(value string) (bool, error) {
	if q.operator != "=" {
		// We can only do this if both parameters are numbers...
		if valNum, errV := strconv.ParseInt(value, 10, 32); errV == nil {
			if queryNum, errQ := strconv.ParseInt(q.value, 10, 32); errQ == nil {
				// both ints, so allow some other comparisons
				if q.operator == ">" {
					return valNum >= queryNum, nil
				} else if q.operator == "<" {
					return valNum <= queryNum, nil
				}
			}
		}

		// Otherwise string compare
		strComp := strings.Compare(value, q.value)
		if q.operator == ">" {
			return strComp > 0, nil
		} else if q.operator == "<" {
			return strComp < 0, nil
		}
		return false, fmt.Errorf("Failed to compare %v, can only use = for values \"%v\", \"%v\"", q.name, q.value, value)
	}

	return q.value == value, nil
}

func (q queryItem) compareSContains(value string) (bool, error) {
	// Make them both lowercase
	lowerValue := strings.ToLower(value)
	lowerQuery := strings.ToLower(q.value)

	return strings.Contains(lowerValue, lowerQuery), nil
}

func (q queryItem) compareSList(value string) (bool, error) {
	if q.operator != "=" {
		return false, fmt.Errorf("Failed to compare, can only use = for value %v", q.name)
	}

	// Expecting a list of items, and we want to compare if we match at least 1 of the list items
	items := strings.Split(q.value, "|")
	for _, item := range items {
		if item == value {
			return true, nil
		}
	}

	return false, nil
}

func parseQueryParams(pathParams map[string]string) ([]queryItem, error) {
	result := []queryItem{}

	for name, value := range pathParams {
		if name == handlers.HostParamName {
			// This one is inserted on our end, not relevant
			continue
		}

		if _, ok := allowedQueryNames[name]; !ok {
			return nil, fmt.Errorf("Search not permitted on field: %v", name)
		}

		// Value is either just a value (if testing equality) or gt,<value> or lt,<value> if > or <
		operator := "="
		if strings.HasPrefix(value, "gt|") {
			operator = ">"
			value = value[3:]
		} else if strings.HasPrefix(value, "lt|") {
			operator = "<"
			value = value[3:]
		} else if strings.HasPrefix(value, "bw|") {
			// Between - so we add a < and then a > here
			// Also the value is 123,456
			bits := strings.Split(value[3:], "|")
			if len(bits) != 2 {
				return nil, fmt.Errorf("Search between did not get 2 values for: %v, got: %v", name, value)
			}

			result = append(result, queryItem{name, ">", bits[0]})

			operator = "<"
			value = bits[1]
		}

		result = append(result, queryItem{name, operator, value})
	}

	return result, nil
}

func matchesSearch(queryParams []queryItem, dataset datasetModel.SummaryFileData) (bool, error) {
	if len(queryParams) <= 0 {
		// No search values specified, allow
		return true, nil
	}

	// Otherwise, we're only returning ones that fit the query criteria
	matchCount := 0
	var match bool
	var err error

	for _, query := range queryParams {
		if query.name == "dataset_id" {
			match, err = query.compareS(dataset.DatasetID)
		} else if query.name == "group_id" {
			match, err = query.compareSList(dataset.Group)
		} else if query.name == "sol" {
			match, err = query.compareSAllowIntConvert(dataset.SOL)
		} else if query.name == "rtt" {
			match, err = query.compareS(dataset.GetRTT())
		} else if query.name == "sclk" {
			match, err = query.compareI(int(dataset.SCLK))
		} else if query.name == "data_file_size" {
			match, err = query.compareI(dataset.DataFileSize)
		} else if query.name == "location_count" {
			match, err = query.compareI(dataset.LocationCount)
		} else if query.name == "dwell_spectra" {
			match, err = query.compareI(dataset.DwellSpectra)
		} else if query.name == "normal_spectra" {
			match, err = query.compareI(dataset.NormalSpectra)
		} else if query.name == "target_id" {
			match, err = query.compareS(dataset.TargetID)
		} else if query.name == "site_id" {
			match, err = query.compareI(int(dataset.SiteID))
		} else if query.name == "drive_id" {
			match, err = query.compareI(int(dataset.DriveID))
		} else if query.name == "detector_config" {
			match, err = query.compareS(dataset.DetectorConfig)
		} else if query.name == "title" {
			// If the title string contains any part of this, we are matched
			match, err = query.compareSContains(dataset.Title)
		} else {
			return false, fmt.Errorf("Cannot compare unknown field: %v", query.name)
		}

		if err != nil {
			return false, err
		}

		// Tally up matches
		if match {
			matchCount++
		}
	}

	// Nothing matched...
	return matchCount == len(queryParams), nil
}

func datasetListing(params handlers.ApiHandlerParams) (interface{}, error) {
	resp := []datasetModel.APIDatasetSummary{}

	// It's a listing request, we don't care about the body...
	s3Path := filepaths.GetDatasetListPath()
	dataSets, err := readDataSetData(params.Svcs, s3Path)
	if err != nil {
		// If error is Not Found, user probably hasn't interacted with this dataset yet, so no need to error out!
		if params.Svcs.FS.IsNotFoundError(err) {
			// Just return empty...
			return &resp, nil
		}
		return nil, err
	}
	//Get user Claims to see if access:breadboard for eg is defined (if this dataset is in group=breadboard)!
	// Unpack the query parameters
	queryParams, err := parseQueryParams(params.PathParams)
	if err != nil {
		return nil, err
	}

	datasetsAuthPath := filepaths.GetDatasetsAuthPath()
	datasetsAuth, err := permission.ReadDatasetsAuth(params.Svcs.FS, params.Svcs.Config.ConfigBucket, datasetsAuthPath)
	if err != nil {
		return nil, err
	}

	userAllowedGroups := permission.GetAccessibleGroups(params.UserInfo.Permissions)

	for _, item := range dataSets.Datasets {
		isPublic, err := permission.CheckAndUpdatePublicDataset(params.Svcs.FS, params.Svcs.Config.ConfigBucket, item.DatasetID, datasetsAuth)
		if err != nil {
			return nil, err
		}
		// Check that the user is allowed to see this dataset based on group permissions
		if !userAllowedGroups[item.Group] && !isPublic {
			continue
		}

		match, err := matchesSearch(queryParams, item)
		if err != nil {
			return nil, err
		}

		if match {
			saveItem := datasetModel.APIDatasetSummary{}

			// make a copy of it to be inserted into the returned structure
			s := item
			s.RTT = s.GetRTT() // For backwards compatibility we can read it as an int, but here we convert to string
			saveItem.SummaryFileData = &s

			saveItem.DataSetLink = params.PathParams[handlers.HostParamName] + "/" + path.Join(datasetPathPrefix, handlers.UrlStreamDownloadIndicator, saveItem.DatasetID, datasetURLEnd)

			if len(saveItem.ContextImage) > 0 {
				saveItem.ContextImageLink = params.PathParams[handlers.HostParamName] + "/" + path.Join(datasetPathPrefix, handlers.UrlStreamDownloadIndicator, saveItem.DatasetID, saveItem.ContextImage)
			} else {
				saveItem.ContextImageLink = ""
			}

			resp = append(resp, saveItem)
		}
	}

	return &resp, nil
}

func datasetFileStream(params handlers.ApiHandlerStreamParams) (*s3.GetObjectOutput, string, string, string, int, error) {
	datasetID := params.PathParams[datasetIdentifier]
	fileName := params.PathParams[idIdentifier]

	// If we're supposed to look at uploaded custom images (which may not already be in the dataset directory)...
	loadCustomType := params.PathParams["loadCustomType"]

	statuscode := 200

	// Due to newly implemented group permissions, we now need to download the dataset summary to check the group is allowable
	summary, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, "", "", "", http.StatusInternalServerError, err
	}

	// We have a few "special" files that can be requested, not by their actual filename, but semantically...
	if fileName == datasetURLEnd {
		// "dataset": we know that the name will be "dataset.bin" because that's what the dataset converter outputs
		fileName = filepaths.DatasetFileName
	} else if fileName == diffractionURLEnd {
		fileName = filepaths.DiffractionDBFileName
	} else if fileName == contextImageReqName || fileName == contextThumbnailReqName || fileName == contextMarkupThumbnailReqName {
		// "context-image": we look up the file name and return that
		// These will eventually return different things (once we modify the dataset converter), but for now to make UI work
		// with different strings, we return the context image for all of them
		fileName = summary.ContextImage
	}

	// Load from dataset directory unless custom loading is requested, where we look up the file in the manual bucket
	imgBucket := params.Svcs.Config.DatasetsBucket
	s3Path := ""
	if len(loadCustomType) <= 0 {
		s3Path = filepaths.GetDatasetFilePath(datasetID, fileName)
	} else {
		s3Path = filepaths.GetCustomImagePath(datasetID, loadCustomType, fileName)
		imgBucket = params.Svcs.Config.ManualUploadBucket
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
					return nil, fileName, *head.ETag, head.LastModified.String(), statuscode, nil
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
					return nil, fileName, *head.ETag, head.LastModified.String(), statuscode, nil
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

	return result, fileName, etag, lm.String(), 0, err
}

// Creation of a dataset - this takes in a POST body that it expects to be a zip file. Query parameter format describes what
// to interpret the POST data to be.
// NOTE: In future we will have to support multiple formats, but for now we only support breadboard MSA files zipped up
// with format set to jpl-breadboard
func datasetCreatePost(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	format := params.PathParams["format"]

	params.Svcs.Log.Debugf("Dataset create started for format: %v, id: %v", datasetID, format)

	// Validate the dataset ID - can't contain funny characters because it ends up as an S3 path
	// NOTE: we also turn space to _ here! Having spaces in the path broke quants because the
	// quant summary file was written with a + instead of a space?!
	datasetID = fileaccess.MakeValidObjectName(datasetID, false)

	if format != "jpl-breadboard" {
		return nil, fmt.Errorf("Unexpected format: \"%v\"", format)
	}

	s3PathStart := path.Join(filepaths.DatasetUploadRoot, datasetID)

	// Check if this exists already...
	existingPaths, err := params.Svcs.FS.ListObjects(params.Svcs.Config.ManualUploadBucket, s3PathStart)
	if err != nil {
		err = fmt.Errorf("Failed to list existing files for dataset ID: %v. Error: %v", datasetID, err)
		params.Svcs.Log.Errorf("%v", err)
		return nil, err
	}

	// If there are any existing paths, we stop here
	if len(existingPaths) > 0 {
		err = fmt.Errorf("Dataset ID already exists: %v", datasetID)
		params.Svcs.Log.Errorf("%v", err)
		return nil, err
	}

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	// Check the contents is just a root dir of .MSA files and NOTHING ELSE
	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	count := 0
	for _, f := range zipReader.File {
		// If the zip path starts with __MACOSX, ignore it, it's garbage that a mac laptop has included...
		//if strings.HasPrefix(f.Name, "__MACOSX") {
		//	continue
		//}

		if f.FileInfo().IsDir() {
			return nil, fmt.Errorf("Zip file must not contain sub-directories. Found: %v", f.Name)
		}

		if !strings.HasSuffix(f.Name, ".msa") {
			return nil, fmt.Errorf("Zip file must only contain MSA files. Found: %v", f.Name)
		}
		count++
	}

	// Make sure it has at least one msa!
	if count <= 0 {
		return nil, errors.New("Zip file did not contain any MSA files")
	}

	// Save the contents as a zip file in the uploads area
	savePath := path.Join(s3PathStart, "spectra.zip")
	err = params.Svcs.FS.WriteObject(params.Svcs.Config.ManualUploadBucket, savePath, body)
	if err != nil {
		return nil, err
	}
	params.Svcs.Log.Debugf("  Wrote: s3://%v/%v", params.Svcs.Config.ManualUploadBucket, savePath)

	// Now save detector info
	savePath = path.Join(s3PathStart, "detector.json")
	detectorFile := datasetArchive.DetectorChoice{
		Detector: "JPL Breadboard",
	}
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ManualUploadBucket, savePath, detectorFile)
	if err != nil {
		return nil, err
	}
	params.Svcs.Log.Debugf("  Wrote: s3://%v/%v", params.Svcs.Config.ManualUploadBucket, savePath)

	// Now save creator info
	savePath = path.Join(s3PathStart, "creator.json")
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ManualUploadBucket, savePath, params.UserInfo)
	if err != nil {
		return nil, err
	}
	params.Svcs.Log.Debugf("  Wrote: s3://%v/%v", params.Svcs.Config.ManualUploadBucket, savePath)

	// Now we trigger a dataset conversion
	result, logId, err := importer.TriggerDatasetReprocessViaSNS(params.Svcs.SNS, params.Svcs.IDGen, datasetID, params.Svcs.Config.DataSourceSNSTopic)
	if err != nil {
		return nil, err
	}

	params.Svcs.Log.Infof("Triggered dataset reprocess via SNS topic. Result: %v. Log ID: %v", result, logId)

	return logId, nil
}
