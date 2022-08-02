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
	"encoding/json"
	"fmt"
	"github.com/pixlise/core/core/notifications"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/pixlise/core/core/logger"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/core/pixlUser"
)

///////////////////////////////////////////////////////////////////////////////////////////
// PIQUANT locally in Docker

type dockerRunner struct {
}

func runDockerInstance(wg *sync.WaitGroup, params PiquantParams, dockerImage string) {
	defer wg.Done()

	// Make a JSON string out of params so it can be passed in
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("Error serialising params for docker instance: %v\n", err)
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
		log.Printf("Running piquant %v in docker failed: %v\n", params.PMCListName, err)
		log.Println(string(out))
		return
	}

	log.Printf("Piquant %v ran successfully:\n", params.PMCListName)
	log.Println(string(out))
}

func (r dockerRunner) runPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, notifications notifications.NotificationManager, creator pixlUser.UserInfo, log logger.ILogger) error {
	// Here we start multiple instances of docker and wait for them all to finish using the WaitGroup
	var wg sync.WaitGroup

	for _, name := range pmcListNames {
		wg.Add(1)

		// Set list name
		params.PMCListName = name

		go runDockerInstance(&wg, params, piquantDockerImage)
	}

	// Wait for all piquant instances to finish
	wg.Wait()

	return nil
}
