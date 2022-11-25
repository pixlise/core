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
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	"github.com/pixlise/core/v2/api/services"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/utils"
)

type QuantListingResponse struct {
	Summaries    []quantModel.JobSummaryItem `json:"summaries"`
	BlessedQuant *quantModel.BlessFileItem   `json:"blessedQuant"`
}

// Lists ALL jobs, regardless of status, creator, etc
// Does this by listing all files in s3://PiquantJobsBucket/RootJobSummaries, then downloads each JSON file and and concats
func quantificationJobAdminList(params handlers.ApiHandlerParams) (interface{}, error) {
	// Firstly, get a list of all JSON files
	items, err := params.Svcs.FS.ListObjects(params.Svcs.Config.PiquantJobsBucket, filepaths.RootJobSummaries+"/")
	if err != nil {
		return []string{}, err
	}

	pathsToRequest := []string{}
	for _, item := range items {
		// This suffix check is redundant (previously was needed, left in just in case). This dir in S3 should ONLY contain job summaries!
		if strings.HasSuffix(item, filepaths.JobSummarySuffix) {
			// Looks like a candidate, check that it has enough bits to extract a dataset ID out of
			bits := strings.Split(item, "/")

			// Looking for: /JobSummaries/<datasetid>-jobs.json
			if len(bits) == 2 {
				pathsToRequest = append(pathsToRequest, item)
			}
		}
	}

	// TODO: Run go routines for each download, combine when all received!

	// Request each file, build a response
	var summaries []quantModel.JobSummaryItem

	for _, qpath := range pathsToRequest {
		var jobsMap quantModel.JobSummaryMap
		err := params.Svcs.FS.ReadJSON(params.Svcs.Config.PiquantJobsBucket, qpath, &jobsMap, false)
		if err != nil {
			// We failed to get the jobs list, so it probably doesn't exist (yet), so just return an empty joblist
			params.Svcs.Log.Errorf("Failed to get job list: s3://%v/%v, jobs for this dataset not included in quant job admin list", params.Svcs.Config.PiquantJobsBucket, qpath)
		} else {
			// read into our summary list
			for _, item := range jobsMap {
				summary := quantModel.SetMissingSummaryFields(item)
				summaries = append(summaries, summary)
			}
		}
	}

	// Update quant summary creator names/emails
	for _, summary := range summaries {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(summary.Params.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (quant admin listing). Error: %v", summary.Params.Creator.UserID, summary.Params.Creator.Name, creatorErr)
		} else {
			summary.Params.Creator = updatedCreator
		}
	}

	sort.Sort(ByJobID(summaries))

	return summaries, nil
}

// ByJobID sorting of quant summaries
type ByJobID []quantModel.JobSummaryItem

func (a ByJobID) Len() int           { return len(a) }
func (a ByJobID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByJobID) Less(i, j int) bool { return a[i].JobID < a[j].JobID }

// Lists jobs for a given dataset
func quantificationList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	summaries, availableQuantIds, err := listQuantsForUser(params.Svcs, datasetID, params.UserInfo.UserID)

	// Also list in-progress quantifications
	processing, err := quantModel.ListQuantJobsForDataset(params.Svcs, params.UserInfo.UserID, params.PathParams[datasetIdentifier])
	if err != nil {
		return nil, err
	}

	// Finally, get the blessed quant ID, so we can send that with the listing
	_, blessItem, _, err := quantModel.GetBlessedQuantFile(params.Svcs, datasetID)
	if err != nil {
		return nil, err
	}

	// Only add quant jobs which don't already have an entry in the available quants
	for _, inProgItem := range processing {
		_, ok := availableQuantIds[inProgItem.JobID]
		if !ok {
			// We only add if ID is not already there
			summaries = append(summaries, quantModel.SetMissingSummaryFields(inProgItem))
		}
	}

	// Update quant summary creator names/emails
	for _, summary := range summaries {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(summary.Params.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (quant user listing). Error: %v", summary.Params.Creator.UserID, summary.Params.Creator.Name, creatorErr)
		} else {
			summary.Params.Creator = updatedCreator
		}
	}

	sort.Sort(ByJobID(summaries))

	// NOTE: only shared quants can be "blessed", so since we modify ids of shared quants to indicate they're shared, we need
	// to modify the blessed quant ID too!
	if blessItem != nil {
		blessItem.JobID = utils.SharedItemIDPrefix + blessItem.JobID
	}

	return &QuantListingResponse{Summaries: summaries, BlessedQuant: blessItem}, nil
}

func listQuantsForUser(svcs *services.APIServices, datasetID string, userID string) ([]quantModel.JobSummaryItem, map[string]bool, error) {
	userQuantSummaryPrefixedPath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.QuantSummaryFilePrefix)
	sharedQuantSummaryPrefixedPath := filepaths.GetSharedQuantPath(datasetID, filepaths.QuantSummaryFilePrefix)

	// List completed quants for user
	userQuants, _ := svcs.FS.ListObjects(svcs.Config.UsersBucket, userQuantSummaryPrefixedPath)
	sharedQuants, _ := svcs.FS.ListObjects(svcs.Config.UsersBucket, sharedQuantSummaryPrefixedPath)

	// Get each summary file and return together (setting shared flag as needed)
	var wg sync.WaitGroup
	expSummaryCount := len(userQuants) + len(sharedQuants)

	summariesCh := make(chan quantModel.JobSummaryItem, expSummaryCount)
	errs := make(chan error, expSummaryCount)

	fetchFunc := func(path string, shared bool) {
		defer wg.Done()

		summary := quantModel.JobSummaryItem{}
		err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, path, &summary, false)
		if err != nil {
			errs <- err
		} else {
			summary.Shared = shared

			if shared {
				summary.JobID = utils.SharedItemIDPrefix + summary.JobID
			}

			summariesCh <- quantModel.SetMissingSummaryFields(summary)
		}
	}

	wg.Add(expSummaryCount)

	for _, item := range userQuants {
		go fetchFunc(item, false)
	}

	for _, item := range sharedQuants {
		go fetchFunc(item, true)
	}

	wg.Wait()
	close(summariesCh)
	close(errs)

	summaries := []quantModel.JobSummaryItem{}
	availableQuantIds := map[string]bool{}

	// Check what we got!
	for err := range errs {
		if err != nil {
			return summaries, availableQuantIds, err
		}
	}

	for summary := range summariesCh {
		summaries = append(summaries, summary)
		availableQuantIds[summary.JobID] = true
	}

	return summaries, availableQuantIds, nil
}

type quantGetResponse struct {
	Summary quantModel.JobSummaryItem `json:"summary"`
	URL     string                    `json:"url"`
}

// NOTE: This returns a summary+link to the quantification download. The actual quant data is returned
// in the download stream handler
func quantificationGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// First, check if the user is allowed to access the given dataset
	datasetID := params.PathParams[datasetIdentifier]

	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, datasetID)
	if err != nil {
		return nil, err
	}

	jobID := params.PathParams[idIdentifier]

	requestJobID := jobID

	// Check if it's a shared one, if so, change our query variables
	summaryFile := filepaths.MakeQuantSummaryFileName(jobID)
	summaryPath := filepaths.GetUserQuantPath(params.UserInfo.UserID, datasetID, summaryFile)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(jobID)
	if isSharedReq {
		jobID = strippedID
		// New job ID!
		summaryFile = filepaths.MakeQuantSummaryFileName(jobID)
		summaryPath = filepaths.GetSharedQuantPath(datasetID, summaryFile)
	}

	summary := quantModel.JobSummaryItem{}
	err = params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, summaryPath, &summary, false)
	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return nil, api.MakeNotFoundError(jobID)
		}
		return nil, err
	}
	// We used to return signed S3 URLs, but these can't be cached (easily)... so now we return from API by streaming from S3...
	url := params.PathParams[handlers.HostParamName] + "/" + path.Join(quantURLPathPrefix, handlers.UrlStreamDownloadIndicator, datasetID, requestJobID)

	summary = quantModel.SetMissingSummaryFields(summary)
	updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(summary.Params.Creator.UserID)
	if creatorErr != nil {
		params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (quant get). Error: %v", summary.Params.Creator.UserID, summary.Params.Creator.Name, creatorErr)
	} else {
		summary.Params.Creator = updatedCreator
	}

	result := quantGetResponse{summary, url}
	return &result, nil
}

func quantificationFileStream(params handlers.ApiHandlerStreamParams) (*s3.GetObjectOutput, string, error) {
	// First, check if the user is allowed to access the given dataset
	datasetID := params.PathParams[datasetIdentifier]

	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, datasetID)
	if err != nil {
		return nil, "", err
	}

	jobID := params.PathParams[idIdentifier]
	fileName := filepaths.MakeQuantDataFileName(jobID)
	binPath := filepaths.GetUserQuantPath(params.UserInfo.UserID, datasetID, fileName)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(jobID)
	if isSharedReq {
		jobID = strippedID
		// New job ID!
		fileName = filepaths.MakeQuantDataFileName(jobID)
		binPath = filepaths.GetSharedQuantPath(datasetID, fileName)
	}

	obj := &s3.GetObjectInput{
		Bucket: aws.String(params.Svcs.Config.UsersBucket),
		Key:    aws.String(binPath),
	}

	result, err := params.Svcs.S3.GetObject(obj)

	return result, fileName, err
}
