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

package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v2/core/logger"
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
	CoresPerNode            int32
	MaxQuantNodes           int32
	QuantNamespace          string
	HotQuantNamespace       string
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

	// Mongo Connection
	MongoEndpoint string
	MongoUsername string
	MongoSecret   string
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
