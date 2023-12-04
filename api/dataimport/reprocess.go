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

package dataimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pixlise/core/v3/api/dataimport/internal/datasetArchive"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
)

// One of the 2 SNS messages we accept. The other is an AWS S3 event message
type datasetReprocessSNSRequest struct {
	DatasetID string `json:"datasetID"`
	JobID     string `json:"jobID"`
}

// Decoding trigger message
// Returns: sourceBucket (optional), sourceFilePath (optional), datasetID, logID
func decodeImportTrigger(triggerMessageBody []byte) (string, string, string, string, error) {
	datasetID := ""

	// job ID to use - we save DB updates about our status using this id
	jobID := ""

	// But if we're being triggered due to new data arriving, these will be filled out
	sourceFilePath := ""
	sourceBucket := ""

	if strings.Contains(string(triggerMessageBody), "\"datasetID\":") {
		// Assume it's a dataset add-on
		var triggerSNS datasetReprocessSNSRequest
		// Work out which kind of trigger it is
		err := json.Unmarshal(triggerMessageBody, &triggerSNS)
		if err != nil {
			return "", "", "", "", fmt.Errorf("Failed to decode dataset reprocess trigger: %v", err)
		}

		// If we have a dataset ID specified, just use that
		if len(triggerSNS.DatasetID) > 0 {
			datasetID = triggerSNS.DatasetID
		} else {
			return "", "", "", "", fmt.Errorf("Failed to find dataset ID in reprocess trigger")
		}

		if len(triggerSNS.JobID) > 0 {
			jobID = triggerSNS.JobID
		} else {
			return "", "", "", "", fmt.Errorf("Failed to find job ID in reprocess trigger")
		}
	} else {
		// Maybe it's a packaged S3 object inside an SNS message
		var snsMsg awsutil.Event
		err := snsMsg.UnmarshalJSON(triggerMessageBody)
		if err != nil {
			return "", "", "", "", fmt.Errorf("Failed to decode dataset import trigger: %v", err)
		}

		if len(snsMsg.Records) < 1 || snsMsg.Records[0].EventSource != "aws:s3" {
			return "", "", "", "", errors.New("Unexpected or no message type embedded in triggering SNS message")
		}

		sourceFilePath = snsMsg.Records[0].S3.Object.Key
		sourceBucket = snsMsg.Records[0].S3.Bucket.Name

		// Based on the file name, we can get a dataset ID
		datasetID, _, err = datasetArchive.DecodeArchiveFileName(sourceFilePath)

		if err != nil {
			// We expected a valid archive file name, if this isn't one, stop here
			return "", "", "", "", fmt.Errorf("Expected archive file, got: %v. Error: %v", sourceFilePath, err)
		}

		// So this is basically a new dataset download, generate a fresh log ID
		jobID = fmt.Sprintf("auto-import-%v (%v)", time.Now().Format("02-Jan-2006 15-04-05"), utils.RandStringBytesMaskImpr(8))
	}

	return sourceBucket, sourceFilePath, datasetID, jobID, nil
}

// Firing a trigger message. Anything calling this is triggering a dataset reimport via a lambda function
func TriggerDatasetReprocessViaSNS(snsSvc awsutil.SNSInterface, jobId string, scanId string, snsTopic string) (*sns.PublishOutput, error) {
	snsReq := datasetReprocessSNSRequest{
		DatasetID: scanId,
		JobID:     jobId,
	}

	snsReqJSON, err := json.Marshal(snsReq)
	if err != nil {
		return nil, errorwithstatus.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to trigger dataset reprocess: %v", err))
	}

	result, err := snsSvc.Publish(&sns.PublishInput{
		Message:  aws.String(string(snsReqJSON)),
		TopicArn: aws.String(snsTopic),
	})

	if err != nil {
		return nil, errorwithstatus.MakeStatusError(http.StatusInternalServerError, fmt.Errorf("Failed to publish SNS topic for dataset regeneration: %v", err))
	}

	return result, nil
}
