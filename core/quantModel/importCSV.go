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
	"fmt"
	"net/http"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/pixlUser"
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
				Command:           "map",
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
