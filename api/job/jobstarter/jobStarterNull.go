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
	"fmt"
	"sync"
	"time"

	"github.com/pixlise/core/v4/core/logger"

	"github.com/pixlise/core/v4/api/config"
	jobrunner "github.com/pixlise/core/v4/api/job/runner"
)

///////////////////////////////////////////////////////////////////////////////////////////
// nullJobStarter for testing

type nullJobStarter struct {
}

func (r *nullJobStarter) StartJob(dockerImage string, jobConfig JobGroupConfig, apiConfig config.APIConfig, requestorUserId string, log logger.ILogger) error {
	namespace := fmt.Sprintf("job-%v", jobConfig.NodeConfig.JobId)

	// Start each container in the namespace
	var wg sync.WaitGroup

	for nodeIdx := 0; nodeIdx < jobConfig.NodeCount; nodeIdx++ {
		wg.Add(1)
		go runNullJob(&wg, jobConfig.GetNodeConfig(nodeIdx), namespace, dockerImage, log)
	}

	// Wait for all job instances to finish
	wg.Wait()

	return nil
}

// This is currently very dumb, we should extend it like the mock s3 backend to mock different failures
// to allow us to test failure modes.
func runNullJob(wg *sync.WaitGroup, jobConfig jobrunner.JobConfig, namespace string, dockerImage string, log logger.ILogger) {
	defer wg.Done()

	fmt.Println("Creating pod...")

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec
	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status

		for i := 1; i < 5; i++ {
			log.Infof("NullJob Loop: " + string(rune(i)))
		}
		time.Sleep(5 * time.Second)
	}
}
