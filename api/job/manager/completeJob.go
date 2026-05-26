package jobmanager

import (
	"context"
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func completeQuantMultiNodeJob(jg *jobconfig.JobGroupConfig, jstatus *protos.JobStatus, svcs *services.APIServices) error {
	jobId := jg.JobGroupId

	// Generate the output path for all generated data files & logs
	quantOutPath := filepaths.GetQuantPath(jg.RequestorUserId, jg.AssociatedScanId, "")

	outputCSVName := ""
	outputCSVBytes := []byte{}
	outputCSV := ""

	jobRoot := filepaths.GetJobDataPath(jg.AssociatedScanId, "", "")
	jobS3Path := filepaths.GetJobDataPath(jg.AssociatedScanId, jobId, "")

	// Gather log files straight away, we want any status updates to include the logs!
	piquantLogList, err := quantification.CopyAllLogs(
		svcs.FS,
		svcs.Log,
		svcs.Config.PiquantJobsBucket,
		jobS3Path,
		svcs.Config.UsersBucket,
		path.Join(quantOutPath, filepaths.MakeQuantLogDirName(jobId)),
		jobId,
	)

	if err != nil {
		svcs.Log.Errorf("Quant job %v copyAllLogs failed: %v", jobId, err)
	}

	// Now we can combine the outputs from all runners
	err = nil

	// Again, if we're in ROI mode, we act differently
	errMsg := ""

	outputFileIdx := -1
	for i, cfg := range jg.NodeConfig.OutputFiles {
		filename := path.Base(cfg.RemotePath)
		if filename == quantification.OutputCSVName {
			outputFileIdx = i
			break
		}
	}

	if outputFileIdx < 0 {
		return fmt.Errorf("Failed to determine quantification output file index for %v", quantification.OutputCSVName)
	}

	if jg.QuantByROI {
		jobCfg := jg.NodeConfig.FlattenJobConfig(0)
		pmcFile := path.Base(jobCfg.OutputFiles[outputFileIdx].RemotePath)
		outputCSV, err = quantification.ProcessQuantROIsToPMCs(svcs.FS, svcs.Config.PiquantJobsBucket, jobS3Path, jg.OutputTitle, pmcFile, jg.Combined, jg.ROIs)
		errMsg = "Error when duplicating quant rows for ROI PMCs"
	} else {
		pmcFiles := []string{}
		for c := uint(0); c < jg.NodeCount; c++ {
			jobCfg := jg.NodeConfig.FlattenJobConfig(c)
			pmcFiles = append(pmcFiles, jobCfg.OutputFiles[outputFileIdx].RemotePath)
		}

		outputCSV, err = quantification.CombineQuantOutputsForResultFilePaths(svcs.FS, svcs.Config.PiquantJobsBucket, jg.OutputTitle, pmcFiles)
		errMsg = "Error when combining quants"
	}
	if err != nil {
		//completeJobState(false, fmt.Sprintf("%v: %v", errMsg, err), "", piquantLogList)
		return fmt.Errorf("%v: %v", errMsg, err)
	}

	outputCSVBytes = []byte(outputCSV)
	outputCSVName = "combined.csv"

	// Save to S3
	csvOutPath := path.Join(jobRoot, jobId, "output", outputCSVName)
	svcs.FS.WriteObject(svcs.Config.PiquantJobsBucket, csvOutPath, outputCSVBytes)

	// Convert to binary format
	binFileBytes, elements, err := quantification.ConvertQuantificationCSV(svcs.Log, outputCSV, []string{"PMC", "SCLK", "RTT", "filename"}, nil, false, "", false)
	if err != nil {
		//completeJobState(false, fmt.Sprintf("Error when converting quant CSV to PIXLISE bin: %v", err), quantOutPath, piquantLogList)
		return fmt.Errorf("Error when converting quant CSV to PIXLISE bin: %v", err)
	}

	// Figure out file paths
	binFilePath := filepaths.GetQuantPath(jg.RequestorUserId, jg.AssociatedScanId, filepaths.MakeQuantDataFileName(jobId))
	csvFilePath := filepaths.GetQuantPath(jg.RequestorUserId, jg.AssociatedScanId, filepaths.MakeQuantCSVFileName(jobId))

	// Save bin quant to S3
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, binFilePath, binFileBytes)
	if err != nil {
		// msg := fmt.Sprintf("Error when uploading converted PIXLISE bin file to s3 at \"s3://%v / %v\": %v", svcs.Config.UsersBucket, binFilePath, err)
		// completeJobState(false, msg, quantOutPath, piquantLogList)
		return fmt.Errorf("Error when uploading converted PIXLISE bin file to s3 at \"s3://%v / %v\": %v", svcs.Config.UsersBucket, binFilePath, err)
	}

	// Save combined CSV to where we have the bin file too
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, csvFilePath, outputCSVBytes)
	if err != nil {
		// Non-job-ending error, can't save the CSV... it means it just won't be available when exporting. Still log error about it
		svcs.Log.Errorf("Failed to upload quant CSV file to s3 at \"s3://%v / %v\": %v", svcs.Config.UsersBucket, csvFilePath, err)
	}

	completeMsg := fmt.Sprintf("Nodes ran: %v", jg.NodeCount)
	now := svcs.TimeStamper.GetTimeNowSec()
	summary := &protos.QuantificationSummary{
		Id:     jobId,
		ScanId: jg.AssociatedScanId,
		//Params:   quantStartSettings,
		Elements: elements,
		Status: &protos.JobStatus{
			JobId:            jobId,
			JobItemId:        jobId,
			Status:           protos.JobStatus_COMPLETE,
			Message:          completeMsg,
			StartUnixTimeSec: jstatus.StartUnixTimeSec,
			EndUnixTimeSec:   uint32(now),
			OutputFilePath:   quantOutPath,
			OtherLogFiles:    piquantLogList,
			Name:             jg.JobName,
			Elements:         jg.ElementList,
			RequestorUserId:  jg.RequestorUserId,
		},
	}

	// If we've got a special import that's done by the internal user, we read the owner entry from DB scan auto share table (ScanAutoShareName)
	ownerItem := wsHelpers.MakeOwnerForWrite(jobId, protos.ObjectType_OT_QUANTIFICATION, jg.RequestorUserId, now)
	if jg.RequestorUserId == sessionuser.PIXLISESystemUserId {
		coll := svcs.MongoDB.Collection(dbCollections.ScanAutoShareName)
		autoShareResult := coll.FindOne(context.TODO(), bson.D{{Key: "_id", Value: jg.RequestorUserId}}, options.FindOne())
		if autoShareResult.Err() != nil {
			svcs.Log.Errorf("Failed to read auto-share info for quantification triggered by %v. Quant won't be shared", jg.RequestorUserId)
		} else {
			autoEntry := &protos.ScanAutoShareEntry{}
			err := autoShareResult.Decode(autoEntry)
			if err != nil {
				svcs.Log.Errorf("Failed to decode auto-share info for quantification triggered by %v: %v", jg.RequestorUserId, err)
			} else {
				svcs.Log.Infof("Found scan auto-share entry for quantification requestor \"%v\". Sharing accordingly.", jg.RequestorUserId)
				ownerItem.Viewers = autoEntry.Viewers
				ownerItem.Editors = autoEntry.Editors
			}
		}
	}

	err = quantification.WriteQuantAndOwnershipToDB(summary, ownerItem, svcs.MongoDB)
	if err != nil {
		//completeJobState(false, fmt.Sprintf("Failed to write quantification and ownership to DB: %v. Id: %v", err, jobId), quantOutPath, piquantLogList)
		return fmt.Errorf("Failed to write quantification and ownership to DB: %v. Id: %v", err, jobId)
	}

	// Report success
	//completeJobState(true, completeMsg, quantOutPath, piquantLogList)
	return nil
}

func completeQuantSingleMapJob(jg *jobconfig.JobGroupConfig, jstatus *protos.JobStatus, svcs *services.APIServices) error {
	/*
		// NOTE: Missing status writes - we only write those for map commands! saveQuantJobStatus quits if it's not a map anyway...

		// Complete writing to the jobs bucket
		// Read the resulting CSV
		jobOutputPath := path.Join(jobDataPath, "output")

		// Make the assumed output path
		piquantOutputPath := path.Join(jobOutputPath, pmcFiles[0]+"_result.csv")

		data, err := svcs.FS.ReadObject(svcs.Config.PiquantJobsBucket, piquantOutputPath)
		if err != nil {
			svcs.Log.Errorf("Failed to read PIQUANT output data from: s3://%v/%v. Error: %v", svcs.Config.PiquantJobsBucket, piquantOutputPath, err)
			outputCSVBytes = []byte{}
			outputCSVName = ""
		} else {
			outputCSVBytes = data
			outputCSVName = quantification.OutputCSVName
		}

		// Save to S3
		csvOutPath := path.Join(jobRoot, r.jobId, "output", outputCSVName)
		svcs.FS.WriteObject(svcs.Config.PiquantJobsBucket, csvOutPath, outputCSVBytes)

		// Map commands are more complicated, where they generate status and summaries, the csv, and the protobuf bin version of the csv, etc
		// but all other commands are far simpler.

		// Clear the previously written files
		csvUserFilePath := filepaths.GetUserLastPiquantOutputPath(r.quantStartSettings.RequestorUserId, userParams.ScanId, userParams.Command, filepaths.QuantLastOutputFileName+".csv")
		userLogFilePath := filepaths.GetUserLastPiquantOutputPath(r.quantStartSettings.RequestorUserId, userParams.ScanId, userParams.Command, filepaths.QuantLastOutputLogName)

		err = svcs.FS.DeleteObject(svcs.Config.UsersBucket, csvUserFilePath)
		if err != nil {
			svcs.Log.Errorf("Failed to delete previous piquant output for command %v from s3://%v/%v. Error: %v", userParams.Command, svcs.Config.UsersBucket, csvUserFilePath, err)
		}
		err = svcs.FS.DeleteObject(svcs.Config.UsersBucket, userLogFilePath)
		if err != nil {
			svcs.Log.Errorf("Failed to delete previous piquant log for command %v from s3://%v/%v. Error: %v", userParams.Command, svcs.Config.UsersBucket, userLogFilePath, err)
		}

		// Upload the results file to the user bucket spot
		if len(outputCSVBytes) > 0 {
			err = svcs.FS.WriteObject(svcs.Config.UsersBucket, csvUserFilePath, outputCSVBytes)

			if err != nil {
				svcs.Log.Errorf("Failed to write output data (length=%v bytes) to user destination path s3://%v/%v", len(outputCSVBytes), svcs.Config.UsersBucket, csvUserFilePath, err)
			}
		}

		// We also write out the log file to the user bucket
		logSourcePath := path.Join(jobDataPath, filepaths.PiquantLogSubdir)
		files, err := svcs.FS.ListObjects(svcs.Config.PiquantJobsBucket, logSourcePath)
		if err != nil {
			svcs.Log.Errorf("Failed to retrieve log files from PIQUANT run from: s3://%v/%v", svcs.Config.PiquantJobsBucket, logSourcePath)
		} else {
			// It should have only ONE log! Anyway, write the first one...
			if len(files) > 0 {
				err := svcs.FS.CopyObject(
					svcs.Config.PiquantJobsBucket,
					files[0],
					svcs.Config.UsersBucket,
					userLogFilePath,
				)

				if err != nil {
					svcs.Log.Errorf("Failed to copy log file: %v://%v to data bucket destination: %v", svcs.Config.PiquantJobsBucket, files[0], userLogFilePath)
				}
			}
		}

		// STOP HERE! Non-map commands are simpler, map commands do a whole bunch more to maintain state files which are picked up
		// by quant listing generation
		r.completeJobState(true, "Wrote Fit output CSV", quantOutPath, piquantLogList)*/
	return nil
}
