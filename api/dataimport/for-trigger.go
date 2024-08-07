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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pixlise/core/v4/api/dataimport/internal/datasetArchive"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	sourceBucket, sourceFilePath, datasetID, jobId, err := decodeImportTrigger(triggerMessage)

	// Report a status so API/users can track what's going on already
	logId := os.Getenv("AWS_LAMBDA_LOG_GROUP_NAME") + "/|/" + os.Getenv("AWS_LAMBDA_LOG_STREAM_NAME")

	ts := timestamper.UnixTimeNowStamper{}
	updateJobState(jobId, protos.JobStatus_STARTING, "Starting importer", logId, db, &ts, log)

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

	updateJobState(jobId, protos.JobStatus_RUNNING, "Importing Files", logId, db, &ts, log)

	importedSummary := &protos.ScanItem{}
	result.WorkingDir, importedSummary, result.WhatChanged, result.IsUpdate, err = ImportDataset(localFS, remoteFS, configBucket, manualBucket, datasetBucket, db, datasetID, log, archived)

	if err != nil {
		result.DatasetID = datasetID
		completeJobState(jobId, false, datasetID, err.Error(), "", []string{}, db, &ts, log)
	} else {
		result.DatasetID = importedSummary.Id
		result.DatasetTitle = importedSummary.Title
		completeJobState(jobId, true, result.DatasetID, "Imported successfully", "", []string{}, db, &ts, log)
	}

	// NOTE: We are now passing this responsibility to the caller, because we're very trusting... And they may want
	// to log something about sending notifications if that happens...
	// Ensure we write everything out...
	//log.Close()

	return result, err
}

func updateJobState(jobId string, status protos.JobStatus_Status, message string, logId string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) {
	// NOTE: We only do this if we're NOT an auto-import job. Those are not triggered by the API
	// so don't need to write job states to DB because nothing would pick it up anyway. It'd fail
	// anyway because the API normally writes the initial job state
	if !strings.HasPrefix(jobId, JobIDAutoImportPrefix) {
		job.UpdateJob(jobId, status, message, logId, db, ts, logger)
	} else {
		logger.Infof("Job %v status: %v. Message: %v", jobId, status, message)
	}
}

func completeJobState(jobId string, success bool, scanId string, message string, outputFilePath string, otherLogFiles []string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) {
	if !strings.HasPrefix(jobId, JobIDAutoImportPrefix) {
		// NOTE: We only do this if we're NOT an auto-import job. Those are not triggered by the API
		// so don't need to write job states to DB because nothing would pick it up anyway. It'd fail
		// anyway because the API normally writes the initial job state
		job.CompleteJob(jobId, success, message, outputFilePath, otherLogFiles, db, ts, logger)
	} else {
		if success {
			logger.Infof("Job %v complete: %v. Output path: %v", jobId, message, outputFilePath)

			// We are an externally (to the API) triggered import, and the DB doesn't yet contain a job status
			// entry for us. Now that we're finished, we write a "complete" state into the DB so any APIs running
			// can pick it up and send notifications as needed
			ctx := context.TODO()
			coll := db.Collection(dbCollections.JobStatusName)

			opt := options.InsertOne()

			status := protos.JobStatus_COMPLETE
			if !success {
				status = protos.JobStatus_ERROR
			}

			now := uint32(ts.GetTimeNowSec())

			jobStatus := &protos.JobStatus{
				JobId:                 jobId,
				Status:                status,
				Message:               message,
				JobItemId:             scanId,
				JobType:               protos.JobStatus_JT_IMPORT_SCAN,
				LogId:                 "",
				StartUnixTimeSec:      0,
				LastUpdateUnixTimeSec: now,
				EndUnixTimeSec:        now,
				OutputFilePath:        outputFilePath,
				OtherLogFiles:         otherLogFiles,
				RequestorUserId:       specialUserIds.PIXLISESystemUserId, // We don't have a requestor ID to write, we're an auto import
			}

			insertResult, err := coll.InsertOne(ctx, jobStatus, opt)
			if err != nil {
				logger.Errorf("%v", err)
			} else {
				// Check that it was inserted
				if insertResult.InsertedID != jobId {
					logger.Errorf("completeJobState expected inserted ID: %v, got: %v", jobId, insertResult.InsertedID)
				} else {
					logger.Infof("completeJobState wrote completed state for externally triggered import: %v", jobId)
				}
			}
		} else {
			logger.Errorf("Job %v Failed: %v", jobId, message)
		}
	}
}
