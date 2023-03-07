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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/pixlise/core/v2/core/notifications"

	"github.com/pixlise/core/v2/core/logger"

	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/core/pixlUser"
)

///////////////////////////////////////////////////////////////////////////////////////////
// PIQUANT locally in Docker

type dockerRunner struct {
}

func runDockerInstance(wg *sync.WaitGroup, params PiquantParams, dockerImage string, log logger.ILogger) {
	defer wg.Done()

	// Make a JSON string out of params so it can be passed in
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		log.Errorf("Error serialising params for docker instance: %v", err)
		return
	}
	paramsStr := string(paramsJSON)

	// Start up docker, give it env vars for AWS access
	// and our JSON param blob too
	cmd := exec.Command("/usr/local/bin/docker",
		"run",
		"--rm",
		"-e", "AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
		"-e", "AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
		"-e", "AWS_DEFAULT_REGION="+os.Getenv("AWS_DEFAULT_REGION"),
		"-e", fmt.Sprintf("QUANT_PARAMS=%v", paramsStr),
		dockerImage,
		"/usr/PIQUANT/PiquantRunner",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Infof("Running piquant %v in docker failed: %v\n", params.PMCListName, err)
		log.Infof(string(out))
		return
	}

	log.Infof("Piquant %v ran successfully:\n", params.PMCListName)
	log.Infof(string(out))
}

func (r *dockerRunner) runPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, notifications notifications.NotificationManager, creator pixlUser.UserInfo, log logger.ILogger) error {
	// Here we start multiple instances of docker and wait for them all to finish using the WaitGroup
	var wg sync.WaitGroup

	// Make sure AWS env vars are available, because that's what we'll be passing to PIQUANT docker container
	if len(os.Getenv("AWS_ACCESS_KEY_ID")) <= 0 || len(os.Getenv("AWS_SECRET_ACCESS_KEY")) <= 0 || len(os.Getenv("AWS_DEFAULT_REGION")) <= 0 {
		txt := "No AWS environment variables defined"
		log.Errorf(txt)
		return errors.New(txt)
	}

	for _, name := range pmcListNames {
		wg.Add(1)

		// Set list name
		params.PMCListName = name

		go runDockerInstance(&wg, params, piquantDockerImage, log)
	}

	// Wait for all piquant instances to finish
	wg.Wait()

	return nil
}
