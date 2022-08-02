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
	"fmt"
	"github.com/pixlise/core/core/notifications"
	"sync"
	"time"

	"github.com/pixlise/core/core/logger"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/core/pixlUser"
)

///////////////////////////////////////////////////////////////////////////////////////////
// NullPiquant for testing

type nullRunner struct {
}

func (r nullRunner) runPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, notifications notifications.NotificationManager, creator pixlUser.UserInfo, log logger.ILogger) error {
	namespace := fmt.Sprintf("job-%v", params.JobID)

	// Start each container in the namespace
	var wg sync.WaitGroup
	for _, name := range pmcListNames {
		wg.Add(1)

		// Set the pmc name so it gets sent to the container
		params.PMCListName = name

		go runNullQuantJob(&wg, params, namespace, piquantDockerImage)
	}

	// Wait for all piquant instances to finish
	wg.Wait()

	return nil
}

// This is currently very dumb, we should extend it like the mock s3 backend to mock different failures
// to allow us to test failure modes.
func runNullQuantJob(wg *sync.WaitGroup, params PiquantParams, namespace string, dockerImage string) {
	defer wg.Done()

	fmt.Println("Creating pod...")

	// Now wait for it to finish
	startUnix := time.Now().Unix()
	maxEndUnix := startUnix + config.KubernetesMaxTimeoutSec
	for currUnix := time.Now().Unix(); currUnix < maxEndUnix; currUnix = time.Now().Unix() {
		// Check kubernetes pod status

		for i := 1; i < 5; i++ {
			fmt.Printf("Loop: " + string(rune(i)))
		}
		time.Sleep(5 * time.Second)
	}
}
