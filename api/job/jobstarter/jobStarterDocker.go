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

package jobstarter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
	"github.com/pixlise/core/v4/core/logger"
)

///////////////////////////////////////////////////////////////////////////////////////////
// Runs job locally in Docker

type dockerJobStarter struct {
}

func runDockerInstance(wg *sync.WaitGroup, config jobrunner.JobConfig, dockerImage string, log logger.ILogger) {
	defer wg.Done()

	// Make a JSON string out of params so it can be passed in
	configJSON, err := json.Marshal(config)
	if err != nil {
		log.Errorf("Error serialising config for docker instance: %v", err)
		return
	}
	configStr := string(configJSON)

	// Start up docker, give it env vars for AWS access
	// and our JSON param blob too
	cmd := exec.Command(dockerCommand,
		"run",
		"--rm",
		"-e", "AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
		"-e", "AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"-e", "AWS_DEFAULT_REGION="+os.Getenv("AWS_DEFAULT_REGION"),
		"-e", fmt.Sprintf("%v=%v", jobrunner.JobConfigEnvVar, configStr),
		dockerImage,
		// <-- Assumed that the runner is the default command and it will pick up what to do from the config env var
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Running job %v in docker failed: %v\n", config.JobId, err)
		log.Infof(string(out))
		return
	}

	log.Infof("Job %v ran successfully:\n", config.JobId)
	log.Infof(string(out))
}

func (r *dockerJobStarter) StartJob(jobDockerImage string, jobConfig JobGroupConfig, apiCfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	// Here we start multiple instances of docker and wait for them all to finish using the WaitGroup
	var wg sync.WaitGroup

	// Make sure AWS env vars are available, because that's what we'll be passing to job docker container
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) <= 0 || len(os.Getenv("AWS_SECRET_ACCESS_KEY")) <= 0 || len(os.Getenv("AWS_DEFAULT_REGION")) <= 0 {
		txt := "No AWS environment variables defined"
		log.Errorf(txt)
		return errors.New(txt)
	}

	for nodeIdx := 0; nodeIdx < jobConfig.NodeCount; nodeIdx++ {
		wg.Add(1)
		go runDockerInstance(&wg, jobConfig.GetNodeConfig(nodeIdx), jobDockerImage, log)
	}

	// Wait for all nodes to finish
	wg.Wait()

	return nil
}
