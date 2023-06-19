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
	"strings"
	"time"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/pixlUser"
)

func setJobStatus(statusobj *JobStatus, status JobStatusValue, message string) {
	statusobj.Status = status
	statusobj.Message = message
	if statusobj.PiquantLogList == nil {
		statusobj.PiquantLogList = []string{}
	}
}

func setJobError(statusobj *JobStatus, message string) {
	statusobj.Status = JobError
	statusobj.Message = message
	if statusobj.PiquantLogList == nil {
		statusobj.PiquantLogList = []string{}
	}

	// Set the time stamp as NOW because NOW is when the error happened
	statusobj.EndUnixTime = time.Now().Unix()
}

// NOTE: message, endUnixTime, outputFilePath, piquantLogList are OPTIONAL!
func saveQuantJobStatus(svcs *services.APIServices, datasetID string, jobName string, status *JobStatus, jobLog logger.ILogger, creator pixlUser.UserInfo) {
	// This is evil, but because all this was written for quant map jobs, and runnin gother PIQUANT commands was an after-thought
	// instead of putting if statements everywhere, to ban all status saving to S3 we check if the quant is a weird one here
	if strings.HasPrefix(status.JobID, "cmd-") {
		jobLog.Infof("PIQUANT command run state: %v. Message: %v", status.Status, status.Message)
		return
	}

	// Log the state change
	level := logger.LogInfo
	if status.Status == JobError {
		level = logger.LogError
		err := quantFailedNotification(jobName, svcs.Notifications, creator.UserID)
		if err != nil {
			svcs.Log.Errorf("Notification dispatch error: %v", err.Error())
		}
	} else if status.Status == JobComplete {
		err := endQuantNotification(jobName, svcs.Notifications, creator)
		if err != nil {
			svcs.Log.Errorf("Notification dispatch error: %v", err.Error())
		}
	}

	jobLog.Printf(level, "Job state: %v. Message: %v", status.Status, status.Message)

	// Save it to S3
	statusPath := filepaths.GetJobStatusPath(datasetID, status.JobID)
	err := svcs.FS.WriteJSON(svcs.Config.PiquantJobsBucket, statusPath, *status)
	if err != nil {
		jobLog.Errorf("Failed to upload status for state: %v", status.Status)
	}
}
