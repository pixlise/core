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
