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

package quantModel

import (
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/fileaccess"
)

// JobSummaryItem all metadata stored for an individual job/quant file (even after it was generated)
type JobSummaryItem struct {
	Shared   bool                              `json:"shared"`
	Params   JobStartingParametersWithPMCCount `json:"params"`
	Elements []string                          `json:"elements"`
	*JobStatus
}

//SetMissingSummaryFields - ensure the fields all exist over time.
func SetMissingSummaryFields(summary JobSummaryItem) JobSummaryItem {
	// Fields were introduced over time after there were many quantifications already in existance, so here
	// we ensure that any older JSONs we read are patched with the new field, because it's a smaller surface area
	// to make sure the API sends out an empty list for this field, than to ensure everything using the API checks
	// for nulls in this field!
	if summary.Elements == nil {
		summary.Elements = []string{}
	}

	if summary.Params.RoiIDs == nil {
		summary.Params.RoiIDs = []string{}
	}

	return summary
}

// JobSummaryMap - saved by job updater lambda function, read by quant listing handler
type JobSummaryMap map[string]JobSummaryItem

// GetJobSummary - Get the summary information from a quant job json file in S3
func GetJobSummary(fs fileaccess.FileAccess, bucket string, userID string, datasetID string, jobID string) (JobSummaryItem, error) {
	jobSummary := JobSummaryItem{}

	// We can assume that any Quant being published will have already been "shared" and available in the ShareUserId directory
	summaryFilePath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.MakeQuantSummaryFileName(jobID))

	err := fs.ReadJSON(bucket, summaryFilePath, &jobSummary, false)
	if err != nil {
		return jobSummary, err
	}
	return jobSummary, nil
}
