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
package quantRunner

import (
	"fmt"

	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/core/logger"
)

// NOTE: these are all static params, the PMC list to process is passed in via the number at the end of the host name

// PiquantParams - Parameters for running piquant, as generated by PIXLISE API
type PiquantParams struct {
	RunTimeEnv        string   `json:"runtimeEnv"`
	JobID             string   `json:"jobId"`
	JobsPath          string   `json:"jobsPath"`
	DatasetPath       string   `json:"datasetPath"`
	DetectorConfig    string   `json:"detectorConfig"`
	Elements          []string `json:"elements"`
	Parameters        string   `json:"parameters"`
	DatasetsBucket    string   `json:"datasetsBucket"`
	ConfigBucket      string   `json:"configBucket"`
	PiquantJobsBucket string   `json:"jobBucket"`
	QuantName         string   `json:"name"`
	PMCListName       string   `json:"pmcListName"`
	Command           string   `json:"command"`
}

type QuantRunner interface {
	// Requires the instance count. If PMC files are named node0001.pmcs to node0003.pmcs, instances
	// should be 4, start with 1-based counting for host names
	RunPiquant(piquantDockerImage string, params PiquantParams, pmcListNames []string, cfg config.APIConfig, requestorUserId string, log logger.ILogger) error
}

func GetQuantRunner(name string) (QuantRunner, error) {
	if name == "docker" {
		return &dockerRunner{}, nil
	} else if name == "kubernetes" {
		return &kubernetesRunner{}, nil
	} else if name == "null" {
		return &nullRunner{}, nil
	}
	return nil, fmt.Errorf("Unknown quant runner: %v", name)
}