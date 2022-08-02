// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package quantModel

import (
	"time"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/logger"
	"github.com/pixlise/core/core/pixlUser"
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
