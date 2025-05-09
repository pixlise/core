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
package jobstarter

import (
	"fmt"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
	"github.com/pixlise/core/v4/core/logger"
)

type JobGroupConfig struct {
	JobGroupId  string
	DockerImage string
	FastStart   bool
	NodeCount   int
	NodeConfig  jobrunner.JobConfig
}

func (jg JobGroupConfig) GetNodeConfig(nodeIdx int) jobrunner.JobConfig {
	nodeCfg := jg.NodeConfig.Copy()
	nodeCfg.JobId = fmt.Sprintf("%v-%v", jg.JobGroupId, nodeIdx)
	return nodeCfg
}

type JobStarter interface {
	StartJob(dockerImage string, jobConfig JobGroupConfig, apiConfig config.APIConfig, requestorUserId string, log logger.ILogger) error
}

func GetJobStarter(name string) (JobStarter, error) {
	if name == "docker" {
		return &dockerJobStarter{}, nil
	} else if name == "kubernetes" {
		return &kubernetesJobStarter{}, nil
	} else if name == "null" {
		return &nullJobStarter{}, nil
	}
	return nil, fmt.Errorf("Unknown job starter: %v", name)
}
