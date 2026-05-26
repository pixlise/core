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

package jobexecutor

import (
	"encoding/json"
	"os"

	"github.com/pixlise/core/v4/api/config"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/core/logger"
)

///////////////////////////////////////////////////////////////////////////////////////////
// Runs job locally by directly calling job runner code. Allows easier debugging
// and consistant (non-race-condition ridden) test writing

type localJobExecutor struct {
	bucketsRootPath string
}

func (r *localJobExecutor) StartJob(jobConfig jobconfig.JobGroupConfig, apiCfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	for nodeIdx := uint(0); nodeIdx < jobConfig.NodeCount; nodeIdx++ {
		r.runJobLocally(jobConfig.NodeConfig.FlattenJobConfig(nodeIdx), log)
	}

	// Wait for all nodes to finish
	return nil
}

func (r *localJobExecutor) runJobLocally(config jobconfig.JobConfig, log logger.ILogger) {
	// Make a JSON string out of params so it can be passed in
	configJSON, err := json.Marshal(config)
	if err != nil {
		log.Errorf("Error serialising config for docker instance: %v", err)
		return
	}

	configStr := string(configJSON)

	// Set it as an env var so the job can pick it up
	os.Setenv(jobconfig.JobConfigEnvVar, configStr)
	/*
	   // NOTE: we're running the job by calling the Go code directly instead of executing it in a docker container. This is primarily for testing
	   // and if we specify a path, it will only look in that path to download from buckets...
	   err = jobrunner.RunJob(r.bucketsRootPath)

	   	if err != nil {
	   		log.Errorf("RunJob failed: %v", err)
	   	}
	*/
}
