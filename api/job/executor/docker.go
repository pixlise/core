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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/logger"
)

///////////////////////////////////////////////////////////////////////////////////////////
// Runs job locally in Docker

type dockerJobExecutor struct {
}

func (r *dockerJobExecutor) StartJob(jobConfig JobGroupConfig, apiCfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	// Here we start multiple instances of docker and wait for them all to finish using the WaitGroup
	var wg sync.WaitGroup

	// Make sure AWS env vars are available, because that's what we'll be passing to job docker container
	awsKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecret := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_DEFAULT_REGION")

	sess, err := awsutil.GetSession()
	if err != nil {
		return err
	}
	v, err := sess.Config.Credentials.Get()
	if err != nil {
		return err
	}
	awsKey = v.AccessKeyID
	awsSecret = v.SecretAccessKey
	if len(awsKey) > 0 && len(awsSecret) > 0 && len(awsRegion) <= 0 {
		awsRegion = "us-east-1"
	}

	/*
		// If we don't have the AWS var stuff, try using the default profile
		if len(awsKey) <= 0 || len(awsSecret) <= 0 || len(awsRegion) <= 0 {
			foundDefault := false
			f, err := os.ReadFile("~/.aws/credentials")
			if err == nil {
				lines := strings.Split(string(f), "\n")
				cutset := " \t\r\n"
				for c, line := range lines {
					if strings.Trim(line, cutset) == "[default]" {
						// Next 2 lines should have what we're after
						for i := range []int{c + 1, c + 2} {
							v := strings.Trim(lines[i], cutset)
							if strings.HasPrefix(v, "aws_access_key_id") {
								pos := strings.Index(v, "=")
								if pos > -1 {
									awsKey = strings.Trim(v[pos+1:], cutset)
								}
							}
							if strings.HasPrefix(v, "aws_secret_access_key") {
								pos := strings.Index(v, "=")
								if pos > -1 {
									awsSecret = strings.Trim(v[pos+1:], cutset)
								}
							}
						}

						foundDefault = len(awsKey) > 0 && len(awsSecret) > 0
						break
					}
				}
			}

			if foundDefault && len(awsRegion) <= 0 {
				awsRegion = "us-east-1"
			}
		}
	*/

	if len(awsKey) <= 0 || len(awsSecret) <= 0 || len(awsRegion) <= 0 {
		txt := "Failed to define AWS variables"
		log.Errorf(txt)
		return errors.New(txt)
	}

	for nodeIdx := 0; nodeIdx < jobConfig.NodeCount; nodeIdx++ {
		wg.Add(1)
		go runDockerInstance(&wg, jobConfig.GetNodeConfig(nodeIdx), jobConfig.DockerImage, awsKey, awsSecret, awsRegion, log)
	}

	// Wait for all nodes to finish
	wg.Wait()

	return nil
}

func runDockerInstance(wg *sync.WaitGroup, config jobrunner.JobConfig, dockerImage string, awsKey string, awsSecret string, awsRegion string, log logger.ILogger) {
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
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-e", "AWS_ACCESS_KEY_ID="+awsKey,
		"-e", "AWS_SECRET_ACCESS_KEY="+awsSecret,
		"-e", "AWS_DEFAULT_REGION="+awsRegion,
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
