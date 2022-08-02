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

package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixlise/core/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Configuration for app

// APIConfig combines env vars and config JSON values
type APIConfig struct {
	AWSBucketRegion         string
	AWSCloudwatchRegion     string
	EnvironmentName         string
	LogLevel                logger.LogLevel
	KubernetesLocation      string
	QuantExecutor           string
	NodeCountOverride       int32
	DockerLoginString       string
	CoresPerNode            int32
	MaxQuantNodes           int32
	QuantNamespace          string
	ElasticURL              string
	ElasticUser             string
	ElasticPassword         string
	SentryEndpoint          string
	Auth0Domain             string
	Auth0ManagementClientID string
	Auth0ManagementSecret   string
	AdminEmails             []string
	DataSourceSNSTopic      string
	QuantDestinationPackage string
	QuantObjectType         string
	PosterImage             string
	KubeConfig              string
	PiquantDockerImage      string

	// Our buckets
	DatasetsBucket     string
	UsersBucket        string
	ConfigBucket       string
	ManualUploadBucket string
	PiquantJobsBucket  string

	// Old buckets
	BuildsBucket              string // Piquant download bucket
	DatasourceArtifactsBucket string // Goes away
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// Init config, loads config params
func Init() (APIConfig, error) {
	// Firstly, read command line arguments
	nodeCountOverride := flag.Int("nodeCountOverride", 0, "Overrides node count for quantification, for testing only")
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Now that we have that, read the env config file from S3
	var cfg APIConfig

	customConfig, ok := os.LookupEnv("CUSTOM_CONFIG")
	if !ok || len(customConfig) <= 0 {
		return cfg, errors.New("No CUSTOM_CONFIG environment variable provided")
	}

	err := json.Unmarshal([]byte(customConfig), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("Failed to parse custom config: %v", err)
	}

	if nodeCountOverride != nil && *nodeCountOverride > 0 {
		cfg.NodeCountOverride = int32(*nodeCountOverride)
	}
	cfg.KubeConfig = *kubeconfig

	return cfg, nil
}
