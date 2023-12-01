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
	"context"
	"fmt"
	"net/http"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
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
	ownerItem, err := wsHelpers.MakeOwnerForWrite(quantId, protos.ObjectType_OT_QUANTIFICATION, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())
	if err != nil {
		return quantId, err
	}

	summary := protos.QuantificationSummary{
		Id:     quantId,
		ScanId: scanId,
		Params: &protos.QuantStartingParameters{
			Name:       quantName,
			DataBucket: hctx.Svcs.Config.DatasetsBucket,
			//ConfigBucket:      svcs.Config.ConfigBucket,
			DatasetPath:       filepaths.GetScanFilePath(scanId, filepaths.DatasetFileName),
			DatasetID:         scanId,
			PiquantJobsBucket: hctx.Svcs.Config.PiquantJobsBucket,
			DetectorConfig:    "",
			Elements:          elements,
			Parameters:        "",
			RunTimeSec:        0,
			CoresPerNode:      0,
			StartUnixTimeSec:  uint32(ownerItem.CreatedUnixSec),
			RequestorUserId:   importUser.Id,
			RoiID:             "",
			ElementSetID:      "",
			PIQUANTVersion:    "N/A",
			QuantMode:         quantModeEnum,
			Comments:          comments,
			RoiIDs:            []string{},
			Command:           "map",
		},
		Elements: elements,
		Status: &protos.JobStatus{
			JobID:          quantId,
			Status:         protos.JobStatus_COMPLETE,
			Message:        csvOrigin + " quantification processed",
			PiquantLogs:    []string{},
			EndUnixTimeSec: uint32(ownerItem.CreatedUnixSec),
			OutputFilePath: quantOutPath,
		},
	}

	ctx := context.TODO()

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return quantId, err
	}
	defer sess.EndSession(ctx)

	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.QuantificationsName).InsertOne(sessCtx, &summary)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem)
		if _err != nil {
			return nil, _err
		}
		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)

	if err != nil {
		return quantId, err
	}

	return quantId, nil
}
