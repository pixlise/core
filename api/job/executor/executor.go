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

// Exposes interfaces and structures required to run PIQUANT in the Kubernetes cluster along with functions
// to access quantification files, logs, results and summaries of quant jobs.
package jobexecutor

import (
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

type JobGroupConfig struct {
	JobGroupId  string                   `bson:"_id,omitempty"` // Job group ID
	JobType     protos.JobStatus_JobType // Job type, mostly for annotation of job state
	DockerImage string                   // Docker image to run in each node
	FastStart   bool                     // May go unused - but could be a way to run it locally on this machine if we know it's a quick job
	NodeCount   uint                     // Node count, because NodeConfig can be asked to retrieve config of each node, but here we know the total
	NodeConfig  job.JobConfig            // Node config sources

	// Job meta-data
	AssociatedScanId string   // Empty if none, or if it's across scans
	JobName          string   // Optional job name, eg used for quants
	ElementList      []string // Optional element list, eg used for quants
	RequestorUserId  string

	// Need configs for:
	// NodeOutputCombining - how to combine the outputs, eg PIQUANT map commands
	// Do we need to write overall job output/logs somewhere?
}

type JobExecutor interface {
	StartJob(jobConfig JobGroupConfig, apiConfig config.APIConfig, requestorUserId string, log logger.ILogger) error
	// TODO:
	// CancelJob
	// GetJobStatus
	// RegisterForJobStatusChange
	// GetJobLogs
}

var localPrefix = "local:"

func MakeLocalExecutor(bucketRootPath string) string {
	return localPrefix + bucketRootPath
}

func GetJobExecutor(name string) (JobExecutor, error) {
	switch name {
	case "docker":
		return &dockerJobExecutor{}, nil
	case "kubernetes":
		return &kubernetesJobExecutor{}, nil
	case "null":
		return &nullJobExecutor{}, nil
	}

	if strings.HasPrefix(name, localPrefix) {
		rootPath := name[len(localPrefix):]
		return &localJobExecutor{
			bucketsRootPath: rootPath,
		}, nil
	}

	return nil, fmt.Errorf("Unknown job executor: %v", name)
}
