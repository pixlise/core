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

package quantRunner

import (
	"fmt"
	"sync"
	"time"

	"github.com/pixlise/core/v4/core/logger"

	"github.com/pixlise/core/v4/api/config"
)

///////////////////////////////////////////////////////////////////////////////////////////
// NullPiquant for testing

type nullRunner struct {
}

func (r *nullRunner) RunPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, requestorUserId string, log logger.ILogger) error {
	namespace := fmt.Sprintf("job-%v", params.JobID)

	// Start each container in the namespace
	var wg sync.WaitGroup
	for _, name := range pmcListNames {
		wg.Add(1)

		// Set the pmc name so it gets sent to the container
		params.PMCListName = name

		go runNullQuantJob(&wg, params, namespace, piquantDockerImage, log)
	}

	// Wait for all piquant instances to finish
	wg.Wait()

	return nil
}

// This is currently very dumb, we should extend it like the mock s3 backend to mock different failures
// to allow us to test failure modes.
func runNullQuantJob(wg *sync.WaitGroup, params PiquantParams, namespace string, dockerImage string, log logger.ILogger) {
	defer wg.Done()

	fmt.Println("Creating pod...")

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec
	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status

		for i := 1; i < 5; i++ {
			log.Infof("NullQuantJob Loop: " + string(rune(i)))
		}
		time.Sleep(5 * time.Second)
	}
}
