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

package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"gitlab.com/pixlise/pixlise-go-api/api/filepaths"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/core/quantModel"
)

// How long to keep showing completed quantification jobs once they're done
const maxTimeSecToShowCompleted = 10 * 60

// The job updater lambda function
// See VERY IMPORTANT further down - this must ONLY be called if files within the filepaths.RootJobStatus dir are changed in the bucket.
//
// This monitors S3 jobs bucket root dir: filepaths.RootJobStatus for a given env for file new/changes/deletes. When triggered, it downloads
// all job status info for that dataset and writes to s3://<jobs bucket>/filepaths.RootJobSummaries/<dataset id>JobSummarySuffix. The API
// for returning quants can then check that file to see if there are any to show for the currently loaded dataset/user

func regenJobSummary(fs fileaccess.FileAccess, jobBucket string, jobRootSearchPath string, log logger.ILogger) error {
	log.Printf(logger.LogInfo, "Fetching objects from %v/%v", jobBucket, jobRootSearchPath)

	theDatasetId, _, err := decodeJobStatusPath(jobRootSearchPath)
	if err != nil {
		log.Errorf("Failed to list jobs in %v/%v: %v", jobBucket, jobRootSearchPath, err)
		return err
	}

	log.Infof("Processing quant jobs for dataset: %v...", theDatasetId)

	paths, err := fs.ListObjects(jobBucket, filepaths.GetJobStatusPath(theDatasetId, "")+"/")
	if err != nil {
		log.Errorf("Failed to list jobs in %v/%v: %v", jobBucket, jobRootSearchPath, err)
		return err
	}

	allJobs := quantModel.JobSummaryMap{}
	nowUnix := time.Now().Unix()

	// Work out what we need to download to build our summary of all jobs...
	const expEnd = filepaths.JobStatusSuffix

	statusPaths := []string{}
	paramPaths := []string{}
	jobIds := []string{}

	for _, thisPath := range paths {
		// Is this a status file?
		endBit := thisPath[len(thisPath)-len(expEnd):]
		if endBit == expEnd {
			// Get job ID
			_, thisJobId, err := decodeJobStatusPath(thisPath)
			if err != nil {
				log.Errorf("Error \"%v\" when decoding path: %v", err, thisPath)
			} else {
				//log.Printf(logger.LogInfo, "Processing job ID: %v", thisJobId)

				statusPaths = append(statusPaths, thisPath)
				paramPaths = append(paramPaths, filepaths.GetJobDataPath(theDatasetId, thisJobId, quantModel.JobParamsFileName))
				jobIds = append(jobIds, thisJobId)
			}
		} else {
			log.Errorf("Unexpected file in status listing: %v. Listing path: %v", thisPath, "s3://"+jobBucket+"/"+jobRootSearchPath)
		}
	}

	var wg sync.WaitGroup

	// Functions to download the 2 required files, which get put in their respective channels
	statusCh := make(chan *quantModel.JobStatus, len(statusPaths))
	paramCh := make(chan *quantModel.JobStartingParametersWithPMCs, len(paramPaths))

	expFiles := len(statusPaths) + len(paramPaths)

	wg.Add(expFiles)

	log.Printf(logger.LogInfo, "Fetching %v status files", len(statusPaths))

	for _, item := range statusPaths {
		fetchStatusFile(&wg, fs, log, jobBucket, item, statusCh)
	}

	log.Printf(logger.LogInfo, "Fetching %v param files", len(paramPaths))

	for _, item := range paramPaths {
		fetchParamFile(&wg, fs, log, jobBucket, item, paramCh)
	}

	wg.Wait()
	close(statusCh)
	close(paramCh)

	// Where we got a pair of non-nils, we can process...
	for c, jobId := range jobIds {
		// Get each file read...
		jobStatus := <-statusCh
		jobParams := <-paramCh

		if jobStatus != nil && jobParams != nil {
			paramPath := paramPaths[c]

			// From the params file we should be able to get the dataset path
			if len(jobParams.DatasetPath) <= 0 {
				log.Errorf("Found empty dataset path in job params file: %v/%v", jobBucket, paramPath)
				continue
			}

			// NOTE:  If it's a completed job, and has an output path, and is too old, stop showing it!
			// NOTE2: This is done here and also in the handler for quant job status - because that's used to retrieve quant info from what we generate
			//        and if we don't get run, the jobs still sit there... that's the final filter
			if jobStatus.Status == "complete" && len(jobStatus.OutputFilePath) > 0 && jobStatus.EndUnixTime != 0 && (nowUnix-jobStatus.EndUnixTime) > maxTimeSecToShowCompleted {
				log.Debugf("Skipping completed job id: %v", jobId)
				continue
			}

			// Store in our list of jobs
			// NOTE: here we're storing a variation of the job starting params... we only store the number of PMCs!
			jobParamSummary := quantModel.MakeJobStartingParametersWithPMCCount(*jobParams)

			item := quantModel.JobSummaryItem{
				Shared:    false, // Anything we have here wouldn't be shared!
				Params:    jobParamSummary,
				Elements:  []string{}, // This job hasn't completed yet, so we don't know what elements it will contain...
				JobStatus: jobStatus,
			}

			log.Debugf("Found job id: \"%v\" status is: %v", jobId, jobStatus.Status)

			item = quantModel.SetMissingSummaryFields(item)

			allJobs[jobId] = item
		}
	}

	// Now save the jobs file
	savePath := filepaths.GetJobSummaryPath(theDatasetId)
	err = fs.WriteJSON(jobBucket, savePath, allJobs)
	if err == nil {
		log.Printf(logger.LogInfo, "Writing %v summary to s3://%v/%v", len(allJobs), jobBucket, savePath)
	}

	return err
}

func fetchStatusFile(wg *sync.WaitGroup, fs fileaccess.FileAccess, log logger.ILogger, bucket string, path string, statusCh chan<- *quantModel.JobStatus) {
	defer wg.Done()

	var jobStatus quantModel.JobStatus

	err := fs.ReadJSON(bucket, path, &jobStatus, false)
	if err != nil {
		log.Errorf("Failed to read job status %v/%v: %v", bucket, path, err)
		statusCh <- nil
		return
	}

	statusCh <- &jobStatus
}

func fetchParamFile(wg *sync.WaitGroup, fs fileaccess.FileAccess, log logger.ILogger, bucket string, path string, paramCh chan<- *quantModel.JobStartingParametersWithPMCs) {
	defer wg.Done()
	var jobParams quantModel.JobStartingParametersWithPMCs

	err := fs.ReadJSON(bucket, path, &jobParams, false)
	if err != nil {
		log.Errorf("Failed to read job params %v/%v: %v", bucket, path, err)
		paramCh <- nil
		return
	} else {
		// New field, may not be set in files
		if jobParams.JobStartingParameters != nil && jobParams.RoiIDs == nil {
			jobParams.RoiIDs = []string{}
		}
	}

	paramCh <- &jobParams
}

// Expecting paths to be some form of: filepaths.RootJobStatus+"/<dataset-id>/<job-id>"+filepaths.JobStatusSuffix
// Returns datasetID, jobID, error
func decodeJobStatusPath(path string) (string, string, error) {
	parts := strings.Split(path, "/")

	// Expecting first element to be the bucket root we should be configured for
	nonEmptyParts := []string{}

	for _, part := range parts {
		if len(part) > 0 {
			nonEmptyParts = append(nonEmptyParts, part)
			if len(nonEmptyParts) == 3 {
				break
			}
		}
	}

	if len(nonEmptyParts) != 3 {
		return "", "", errors.New("Failed to parse path: " + path)
	}

	if nonEmptyParts[0] != filepaths.RootJobStatus {
		return "", "", errors.New("Unexpected start to monitoring path: " + nonEmptyParts[0] + ", full path path: " + path)
	}

	// Third part should be suffixed...
	if nonEmptyParts[2] == filepaths.JobStatusSuffix || !strings.HasSuffix(nonEmptyParts[2], filepaths.JobStatusSuffix) {
		return "", "", errors.New("Unexpected file name in path: " + nonEmptyParts[2] + ", full path path: " + path)
	}

	// Return just the dataset id part
	datasetID := nonEmptyParts[1]
	jobID := nonEmptyParts[2][0 : len(nonEmptyParts[2])-len(filepaths.JobStatusSuffix)]

	if len(datasetID) <= 0 {
		return "", "", errors.New("Dataset ID found is invalid from path: " + path)
	}
	if len(jobID) <= 0 {
		return "", "", errors.New("Job ID found is invalid from path: " + path)
	}

	return datasetID, jobID, nil
}

type s3PathInfo struct {
	bucket string
	path   string
}

// VERY IMPORTANT: This lambda function MUST be configured to only be called for changes in the
// filepaths.RootJobStatus (as in whats in the const var!) directory in the bucket. It does listings on
// that directory, and writes to filepaths.RootJobSummaries, so you don't want this to recursibely get
// called because it's monitoring the wrong root path!
func handler(ctx context.Context, s3Event events.S3Event) error {
	sess, err := awsutil.GetSession()
	if err != nil {
		return err
	}
	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		return err
	}
	fs := fileaccess.MakeS3Access(s3svc)

	// We have to write to stdout so it gets to cloudwatch logs via lambda magic
	stdLog := logger.StdOutLogger{}

	/*
	   NOTE: S3 event at this point is:
	   {
	   	Records:[
	   		{
	   			EventVersion:2.1
	   			EventSource:aws:s3
	   			AWSRegion:us-east-1
	   			EventTime:2020-08-24 01:20:42.655 +0000 UTC
	   			EventName:ObjectCreated:Put
	   			PrincipalID:{PrincipalID:AWS:AIDA6AOWGDOHF37MOKWLS}
	   			RequestParameters:{SourceIPAddress:3.12.95.94}
	   			ResponseElements: map[
	   				x-amz-id-2:7nADdXJ3v1i9OqornwAZVx8gU8tPTMH6nW03bYH7mtBU2rm77+uhQsRfz/rqJ/JpGw0y9AvFuF+kZJ5jdetnnEAgS3zeVvOU
	   				x-amz-request-id:1QDP9Q2V1V2SBZ8J
	   				]
	   			S3: {
	   				SchemaVersion:1.0
	   				ConfigurationID:a0abaa17-dd0a-47ea-9271-824260374c67
	   				Bucket: {
	   					Name:devstack-persistencepiquantjobs65c7175e-1dg51nw1ye1rk
	   					OwnerIdentity:{PrincipalID:AP902Y0PI20DF}
	   					Arn:arn:aws:s3:::devstack-persistencepiquantjobs65c7175e-1dg51nw1ye1rk
	   				}
	   				Object: {
	   					Key:JobStatus/983561/u9a8tr7ja02qu2m0-status.json
	   					Size:2236
	   					URLDecodedKey:
	   					VersionID:
	   					ETag:d8220bef7b762d33dcf3a222586e4fc7
	   					Sequencer:005F4315ED64D92A1B
	   				}
	   			}
	   		}
	   	]
	   }

	   We need to generate a unique list of root paths we're going to execute for
	*/
	uniqueRoots := map[string]s3PathInfo{}

	// Run through the incoming S3 event and regen job summary for any buckets mentioned
	for _, record := range s3Event.Records {
		s3ev := record.S3
		//stdLog.Infof("Adding trigger S3 record: s3://%v/%v\n", s3ev.Bucket.Name, s3ev.Object.Key)

		keyBits := strings.Split(s3ev.Object.Key, "/")
		if len(keyBits) > 2 {
			uniqueRoots[path.Join(s3ev.Bucket.Name, keyBits[0], keyBits[1], keyBits[2])] = s3PathInfo{s3ev.Bucket.Name, path.Join(keyBits[0], keyBits[1])}
		}
		uniqueRoots[path.Join(s3ev.Bucket.Name, s3ev.Object.Key)] = s3PathInfo{s3ev.Bucket.Name, s3ev.Object.Key}
	}

	// Now execute for each unique path
	errCount := 0
	for _, s3Path := range uniqueRoots {
		err := regenJobSummary(fs, s3Path.bucket, s3Path.path, stdLog)
		if err != nil {
			// Don't stop here!
			stdLog.Errorf("Regen FAILED for path: s3://%v/%v. Error: %v.", s3Path.bucket, s3Path.path, err)
			errCount++
		}
	}

	if errCount > 0 {
		return fmt.Errorf("Regen failed for %v paths", errCount)
	}

	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())
	lambda.Start(handler)
	/*	sess, _ := awsutil.GetSession()
		s3svc, _ := awsutil.GetS3(sess)
		stdLog := logger.StdOutLogger{}
		regenJobSummary(s3svc, "devstack-persistencepiquantjobs65c7175e-1dg51nw1ye1rk", filepaths.RootJobStatus, stdLog)*/
}
