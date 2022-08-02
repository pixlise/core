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
	"fmt"
	"net/http"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/pixlUser"
)

func ImportQuantCSV(svcs *services.APIServices, datasetID string, importUser pixlUser.UserInfo, csvBody string, csvOrigin string, idPrefix string, quantName string, quantModeEnum string, comments string) (string, error) {
	// We create a pretend job number so we have an ID for this quantification. Don't want it to match any others
	jobID := idPrefix + "_" + svcs.IDGen.GenObjectID()

	// We can now convert the CSV to a quantification bin file
	binFileBytes, elements, err := ConvertQuantificationCSV(svcs.Config.EnvironmentName, csvBody, []string{"PMC", "SCLK", "RTT", "filename"}, "", false, "", false)
	if err != nil {
		return jobID, api.MakeBadRequestError(err)
	}

	// Figure out file names
	quantOutPath := filepaths.GetUserQuantPath(importUser.UserID, datasetID, "")

	binFilePath := filepaths.GetUserQuantPath(importUser.UserID, datasetID, filepaths.MakeQuantDataFileName(jobID))
	summaryFilePath := filepaths.GetUserQuantPath(importUser.UserID, datasetID, filepaths.MakeQuantSummaryFileName(jobID))
	csvFilePath := filepaths.GetUserQuantPath(importUser.UserID, datasetID, filepaths.MakeQuantCSVFileName(jobID))

	// Save bin quant to S3
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, binFilePath, binFileBytes)
	if err != nil {
		return jobID, api.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to upload %v quantification to S3", csvOrigin))
	}

	// If we've come this far, we can't really "fail" anymore, we just have files missing...
	// Save CSV to S3
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, csvFilePath, []byte(csvBody))
	if err != nil {
		svcs.Log.Errorf("Failed to upload source CSV for %v quantification: %v", csvOrigin, jobID)
	}

	// Finally, output a "summary" file to go with the quant, so API can quickly load up its metadata
	timeNow := svcs.TimeStamper.GetTimeNowSec()
	summary := JobSummaryItem{
		Shared: false,
		Params: JobStartingParametersWithPMCCount{
			PMCCount: 0,
			JobStartingParameters: &JobStartingParameters{
				Name:       quantName,
				DataBucket: svcs.Config.DatasetsBucket,
				//ConfigBucket:      svcs.Config.ConfigBucket,
				DatasetPath:       filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetFileName),
				DatasetID:         datasetID,
				PiquantJobsBucket: svcs.Config.PiquantJobsBucket,
				DetectorConfig:    "",
				Elements:          elements,
				Parameters:        "",
				RunTimeSec:        0,
				CoresPerNode:      0,
				StartUnixTime:     timeNow,
				Creator:           importUser,
				RoiID:             "",
				ElementSetID:      "",
				PIQUANTVersion:    "N/A",
				QuantMode:         quantModeEnum,
				Comments:          comments,
				RoiIDs:            []string{},
			},
		},
		Elements: elements,
		JobStatus: &JobStatus{
			JobID:          jobID,
			Status:         JobComplete,
			Message:        csvOrigin + " quantification processed",
			PiquantLogList: []string{},
			EndUnixTime:    timeNow,
			OutputFilePath: quantOutPath,
		},
	}

	if err == nil {
		err = svcs.FS.WriteJSON(svcs.Config.UsersBucket, summaryFilePath, summary)
	}

	if err != nil {
		svcs.Log.Errorf("Failed to create/upload job summary for %v quantification: %v", csvOrigin, jobID)
	}

	return jobID, nil
}
