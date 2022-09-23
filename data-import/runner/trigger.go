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

package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/utils"
	"github.com/pixlise/core/v2/data-import/internal/importerutils"
)

type datasetAddonData struct {
	Dir string `json:"dir"`
	Log string `json:"log"`
}

type datasetAddonTrigger struct {
	DatasetAddons datasetAddonData `json:"datasetaddons"`
}

func decodeImportTrigger(triggerMessageBody []byte) (string, string, string, string, error) {
	datasetID := ""

	// Log ID to use - this forms part of the log stream in cloudwatch
	logID := ""

	// But if we're being triggered due to new data arriving, these will be filled out
	sourceFilePath := ""
	sourceBucket := ""

	if strings.Contains(string(triggerMessageBody), "\"datasetaddons\":") {
		// Assume it's a dataset add-on
		var datasetAddon datasetAddonTrigger
		// Work out which kind of trigger it is
		err := json.Unmarshal(triggerMessageBody, &datasetAddon)
		if err != nil {
			return "", "", "", "", fmt.Errorf("Failed to decode dataset addon trigger: %v", err)
		}

		// It's just a dataset reprocess request, read the dataset ID that's being requested
		// NOTE: Path here is something like /dataset-addons/<datasetID>/custom_meta.json
		// So we need the middle parth of the path
		parts := strings.Split(datasetAddon.DatasetAddons.Dir, "/")
		datasetID = parts[1]
		if len(parts) != 3 || len(datasetID) <= 1 {
			return "", "", "", "", fmt.Errorf("Failed to find dataset ID from path: %v", datasetAddon.DatasetAddons.Dir)
		}
		logID = datasetAddon.DatasetAddons.Log
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
		datasetID, _, err = importerutils.DecodeArchiveFileName(sourceFilePath)

		if err != nil {
			// We expected a valid archive file name, if this isn't one, stop here
			return "", "", "", "", fmt.Errorf("Expected archive file, got: %v. Error: %v", sourceFilePath, err)
		}

		// So this is basically a new dataset download, generate a fresh log ID
		logID = fmt.Sprintf("auto-import-%v (%v)", time.Now().Format("02-Jan-2006 15-04-05"), utils.RandStringBytesMaskImpr(8))
	}

	return sourceBucket, sourceFilePath, datasetID, logID, nil
}
