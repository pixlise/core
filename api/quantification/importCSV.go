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

package quantification

import (
	"fmt"
	"net/http"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func ImportQuantCSV(
	hctx wsHelpers.HandlerContext,
	scanId string,
	importUser *protos.UserInfo,
	csvBody string,
	csvOrigin string,
	idPrefix string,
	quantName string,
	quantModeEnum string,
	comments string) (string, error) {
	// We create a pretend job number so we have an ID for this quantification. Don't want it to match any others
	quantId := idPrefix + "_" + hctx.Svcs.IDGen.GenObjectID()

	// We can now convert the CSV to a quantification bin file
	binFileBytes, elements, err := ConvertQuantificationCSV(hctx.Svcs.Log, csvBody, []string{"PMC", "SCLK", "RTT", "filename"}, nil, false, "", false)
	if err != nil {
		return quantId, errorwithstatus.MakeBadRequestError(err)
	}

	// Figure out file names
	quantOutPath := filepaths.GetQuantPath(importUser.Id, scanId, "")

	binFilePath := filepaths.GetQuantPath(importUser.Id, scanId, filepaths.MakeQuantDataFileName(quantId))
	csvFilePath := filepaths.GetQuantPath(importUser.Id, scanId, filepaths.MakeQuantCSVFileName(quantId))

	// Save bin quant to S3
	err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.UsersBucket, binFilePath, binFileBytes)
	if err != nil {
		return quantId, errorwithstatus.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to upload %v quantification to S3", csvOrigin))
	}

	// If we've come this far, we can't really "fail" anymore, we just have files missing...
	// Save CSV to S3
	err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.UsersBucket, csvFilePath, []byte(csvBody))
	if err != nil {
		hctx.Svcs.Log.Errorf("Failed to upload source CSV for %v quantification: %v", csvOrigin, quantId)
	}

	// Finally, write the summary data to DB along with ownership entry
	ownerItem := wsHelpers.MakeOwnerForWrite(quantId, protos.ObjectType_OT_QUANTIFICATION, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())

	summary := protos.QuantificationSummary{
		Id:     quantId,
		ScanId: scanId,
		Params: &protos.QuantStartingParameters{
			UserParams: &protos.QuantCreateParams{
				Command:        "map",
				Name:           quantName,
				ScanId:         scanId,
				Pmcs:           []int32{},
				Elements:       elements,
				DetectorConfig: "",
				Parameters:     "",
				RunTimeSec:     0,
				QuantMode:      quantModeEnum,
				RoiIDs:         []string{},
				IncludeDwells:  false,
			},
			PmcCount:          0,
			ScanFilePath:      filepaths.GetScanFilePath(scanId, filepaths.DatasetFileName),
			DataBucket:        hctx.Svcs.Config.DatasetsBucket,
			PiquantJobsBucket: hctx.Svcs.Config.PiquantJobsBucket,
			CoresPerNode:      0,
			StartUnixTimeSec:  uint32(ownerItem.CreatedUnixSec),
			RequestorUserId:   importUser.Id,
			PIQUANTVersion:    "N/A",
			Comments:          comments,
		},
		Elements: elements,
		Status: &protos.JobStatus{
			JobId:          quantId,
			Status:         protos.JobStatus_COMPLETE,
			Message:        csvOrigin + " quantification processed",
			OtherLogFiles:  []string{},
			EndUnixTimeSec: uint32(ownerItem.CreatedUnixSec),
			OutputFilePath: quantOutPath,
		},
	}

	err = writeQuantAndOwnershipToDB(&summary, ownerItem, hctx.Svcs.MongoDB)
	if err != nil {
		return quantId, err
	}

	return quantId, nil
}
