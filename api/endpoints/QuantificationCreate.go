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
	"path"
	"strings"
	"sync"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/quantModel"
)

func quantificationPost(params handlers.ApiHandlerParams) (interface{}, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req quantModel.JobCreateParams
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, err
	}

	if len(req.Command) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("PIQUANT command to run was not supplied"))
	}

	// We only require the name to be set in map mode
	if req.Command == "map" {
		if len(req.Name) <= 0 {
			return nil, api.MakeBadRequestError(errors.New("Name not supplied"))
		}

		// Validate things, eg no quants named the same already, parameters filled out as expected, etc...
		if quantModel.CheckQuantificationNameExists(req.Name, params.PathParams[datasetIdentifier], params.UserInfo.UserID, params.Svcs) {
			return nil, api.MakeBadRequestError(fmt.Errorf("Name already used: %v", req.Name))
		}
	} else {
		req.Name = ""
	}

	// Might be given either empty elements, or if string conversion (with split(',')) maybe we got [""]...
	if len(req.Elements) <= 0 || len(req.Elements[0]) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("Elements not supplied"))
	}

	if len(req.DetectorConfig) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("DetectorConfig not supplied"))
	}

	// At this point, we're assuming that the detector config is a valid config name / version. We need this to be the path of the config in S3
	// so here we convert it and ensure it's valid
	detectorConfigBits := strings.Split(req.DetectorConfig, "/")
	if len(detectorConfigBits) != 2 || len(detectorConfigBits[0]) < 0 || len(detectorConfigBits[1]) < 0 {
		return nil, api.MakeBadRequestError(errors.New("DetectorConfig not in expected format"))
	}

	// Form the string
	// NOTE: we would want to use this:
	// req.DetectorConfig = filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], "")
	// But can't because then the root "/DetectorConfig" is added twice!
	req.DetectorConfig = path.Join(detectorConfigBits[0], filepaths.PiquantConfigSubDir, detectorConfigBits[1])

	/*
		if !isValidConfigName(req.DetectorConfig) {
			return nil, api.MakeBadRequestError(errors.New("DetectorConfig not found"))
		}
	*/
	// Parameters can be empty, what could we validate here?

	if req.RunTimeSec < 1 {
		return nil, api.MakeBadRequestError(errors.New("RunTimeSec is invalid"))
	}

	// Can't check this! ROI being empty implies "whole dataset", we don't have a special string defined for this.
	//	if len(req.RoiID) <= 0 {
	//		return nil, api.MakeBadRequestError(errors.New("ROI ID not supplied"))
	//	}

	// Set these locally
	req.Creator = params.UserInfo
	req.DatasetID = params.PathParams[datasetIdentifier]
	req.DatasetPath = filepaths.GetDatasetFilePath(req.DatasetID, filepaths.DatasetFileName)

	// Went unused...
	req.ElementSetID = ""

	// New field, may not be set
	if req.RoiIDs == nil {
		req.RoiIDs = []string{}
	}

	var wg sync.WaitGroup
	jobID, err := quantModel.CreateJob(params.Svcs, req, &wg)

	if err != nil {
		return jobID, err
	}

	// If it's NOT a map command, we wait around for the result and pass it back in the response
	// but for map commands, we just pass back the generated job id instantly
	if req.Command == "map" {
		// TODO: Use quantificationCreateResponse
		// TODO: Use Location header and return 202 (Accepted) for job creation
		// See: https://farazdagi.com/2014/rest-and-long-running-jobs/
		// Talks about being RFC 7231 compliant

		return jobID, nil
	}

	// Wait around for the output file to appear, or for the job to end up in an error state
	wg.Wait()

	// Return error or the resulting CSV, whichever happened
	userOutputFilePath := filepaths.GetUserLastPiquantOutputPath(req.Creator.UserID, req.DatasetID, req.Command, filepaths.QuantLastOutputFileName+".csv")
	bytes, err := params.Svcs.FS.ReadObject(params.Svcs.Config.UsersBucket, userOutputFilePath)
	if err != nil {
		return nil, errors.New("PIQUANT command: " + req.Command + " failed.")
	}

	return string(bytes), nil
}
