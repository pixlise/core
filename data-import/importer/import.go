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

package importer

import (
	"fmt"

	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	dataConverter "github.com/pixlise/core/v2/data-import/data-converter"
	datasetArchive "github.com/pixlise/core/v2/data-import/dataset-archive"
)

func ImportForTrigger(triggerMessage []byte, envName string, configBucket string, datasetBucket string, manualBucket string, log logger.ILogger) error {
	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger(triggerMessage)

	if err != nil {
		return err
	}

	// Initialise stuff
	sess, err := awsutil.GetSession()
	if err != nil {
		return err
	}

	if log == nil {
		log, err = logger.InitCloudWatchLogger(sess, "/dataset-importer/"+envName, datasetID+"-"+logID, logger.LogDebug, 30, 3)
		if err != nil {
			return err
		}
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		return err
	}

	localFS := &fileaccess.FSAccess{}
	remoteFS := fileaccess.MakeS3Access(svc)

	// Check if we were triggered via a new file arriving, if so, archive it
	archived := false
	if len(sourceBucket) > 0 && len(sourceFilePath) > 0 {
		err = datasetArchive.AddToDatasetArchive(remoteFS, log, datasetBucket, sourceBucket, sourceFilePath)
		if err != nil {
			err = fmt.Errorf("Failed to archive incoming file: \"s3://%v/%v\"", sourceBucket, sourceFilePath)
			log.Errorf("%v", err)
			return err
		}
		archived = true
	} else if len(sourceBucket) > 0 || len(sourceFilePath) > 0 {
		// We need BOTH to be set to something for this to work, only one of them is set
		err = fmt.Errorf("Trigger message must specify bucket AND path, received bucket=%v, path=%v", sourceBucket, sourceFilePath)
		log.Errorf("%v", err)
		return err
	}

	_, err = dataConverter.ImportDataset(localFS, remoteFS, configBucket, manualBucket, datasetBucket, datasetID, log, archived)

	return err
}
