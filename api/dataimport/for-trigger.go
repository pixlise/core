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

// Implements importer triggering based on SNS queues. This decodes incoming SNS messages and extracts files ready
// for importer code to run
package dataimport

import (
	"fmt"

	"github.com/pixlise/core/v3/api/dataimport/internal/datasetArchive"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

// Structure returned after importing
// NOTE: the logger must have Close() called on it, otherwise we may lose the last few log events
type ImportResult struct {
	WorkingDir   string         // so it can be cleaned up by caller if needed
	WhatChanged  string         // what changed between this import and a previous one, for notification reasons
	IsUpdate     bool           // IsUpdate flag
	DatasetTitle string         // Name of the dataset that was imported
	DatasetID    string         // ID of the dataset that was imported
	Logger       logger.ILogger // Caller must call Close() on it, otherwise we may lose the last few log events
}

// ImportForTrigger - Parses a trigger message (from SNS) and decides what to import
// Returns:
// Result struct - NOTE: logger must have Close() called on it, otherwise we may lose the last few log events
// Error (or nil)
func ImportForTrigger(
	triggerMessage []byte,
	envName string,
	configBucket string,
	datasetBucket string,
	manualBucket string,
	db *mongo.Database,
	log logger.ILogger,
	remoteFS fileaccess.FileAccess) (ImportResult, error) {
	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger(triggerMessage)

	result := ImportResult{
		WorkingDir:   "",
		WhatChanged:  "",
		IsUpdate:     false,
		DatasetTitle: "",
		DatasetID:    "",
	}

	if err != nil {
		return result, err
	}

	// Initialise stuff
	sess, err := awsutil.GetSession()
	if err != nil {
		return result, err
	}

	if log == nil {
		log, err = logger.InitCloudWatchLogger(sess, "/dataset-importer/"+envName, datasetID+"-"+logID, logger.LogDebug, 30, 3)
		if err != nil {
			return result, err
		}
	}

	// Return the logger...
	result.Logger = log

	// Handle panics. Here we close the logger if there is a panic, to ensure anything we have is written out to cloudwatch!
	defer logger.HandlePanicWithLog(log)

	localFS := &fileaccess.FSAccess{}

	// Check if we were triggered via a new file arriving, if so, archive it
	archived := false
	if len(sourceBucket) > 0 && len(sourceFilePath) > 0 {
		exists, err := datasetArchive.AddToDatasetArchive(remoteFS, log, datasetBucket, sourceBucket, sourceFilePath)
		if err != nil {
			log.Errorf("%v", err)
			return result, err
		}

		if exists {
			// This file existed already in our archive, so it must've been processed already and we have nothing to do
			// NOTE: This condition exists because the pipeline seems to deliver the same zip file multiple times
			log.Infof("File already exists in archive, processing stopped. File was: \"%v\"", sourceFilePath)

			// Not an error, we just consider ourselves succesfully complete now
			return result, err
		}

		archived = true
	} else if len(sourceBucket) > 0 || len(sourceFilePath) > 0 {
		// We need BOTH to be set to something for this to work, only one of them is set
		err = fmt.Errorf("Trigger message must specify bucket AND path, received bucket=%v, path=%v", sourceBucket, sourceFilePath)
		log.Errorf("%v", err)
		return result, err
	}

	importedSummary := &protos.ScanItem{}
	result.WorkingDir, importedSummary, result.WhatChanged, result.IsUpdate, err = ImportDataset(localFS, remoteFS, configBucket, manualBucket, datasetBucket, db, datasetID, log, archived)
	result.DatasetID = importedSummary.Id
	result.DatasetTitle = importedSummary.Title

	if err != nil {
		log.Errorf("%v", err)
	}

	// NOTE: We are now passing this responsibility to the caller, because we're very trusting... And they may want
	// to log something about sending notifications if that happens...
	// Ensure we write everything out...
	//log.Close()

	return result, err
}
