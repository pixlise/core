// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package endpoints

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/quantModel"
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

	// Validate things, eg no quants named the same already, parameters filled out as expected, etc...
	if quantModel.CheckQuantificationNameExists(req.Name, params.PathParams[datasetIdentifier], params.UserInfo.UserID, params.Svcs) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Name already used: %v", req.Name))
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

	jobID, err := quantModel.CreateJob(params.Svcs, req, true)

	if err != nil {
		return jobID, err
	}

	// TODO: Use quantificationCreateResponse
	// TODO: Use Location header and return 202 (Accepted) for job creation
	// See: https://farazdagi.com/2014/rest-and-long-running-jobs/
	// Talks about being RFC 7231 compliant

	return jobID, nil
}
