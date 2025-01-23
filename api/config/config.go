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

// API configuration as read from strings/JSON and some constants defined here also
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Configuration for app

// APIConfig combines env vars and config JSON values
type APIConfig struct {
	AdminEmails []string

	Auth0Domain             string
	Auth0ManagementClientID string
	Auth0ManagementSecret   string
	Auth0ClientSecret       string
	Auth0NewUserRoleID      string

	ConfigBucket string

	CoresPerNode int32

	DataSourceSNSTopic string
	CoregSqsQueueUrl   string

	DatasetsBucket string

	EnvironmentName string

	HotQuantNamespace string // Used for faster PIQUANT runs, eg executing a spectral fit

	KubernetesLocation string // "internal" vs "external"

	LogLevel           logger.LogLevel // Can be changed at runtime, but if API restarts, it goes back to configured value
	ManualUploadBucket string

	// Mongo Connection
	MongoSecret string

	PiquantDockerImage string // PIQUANT docker image to use to run a job
	PiquantJobsBucket  string // PIQUANT job scratch drive

	PosterImage             string
	QuantDestinationPackage string

	QuantExecutor  string
	QuantNamespace string // Used for running large multi-node quants

	QuantObjectType string

	SentryEndpoint string

	UsersBucket string

	ZenodoURI         string
	ZenodoAccessToken string

	// Vars not set by environment
	NodeCountOverride      int32
	QuantNodeMaxRuntimeSec int32
	MaxQuantNodes          int32
	KubeConfig             string // Env sets this via command line parameter

	// Web Socket config
	WSWriteWaitMs       uint
	WSPongWaitMs        uint
	WSPingPeriodMs      uint
	WSMaxMessageSize    uint
	WSMessageBufferSize uint

	// Local file caching (from S3 to where API is running)
	MaxFileCacheAgeSec    uint
	MaxFileCacheSizeBytes uint

	ImportJobMaxTimeSec  uint32
	PIQUANTJobMaxTimeSec uint32

	// The GroupId of the group a new user is added to by default as a member
	DefaultUserGroupId string

	// PIXLISE backup & restore settings
	DataBackupBucket string
	BackupEnabled    bool
	RestoreEnabled   bool

	ImpersonateEnabled bool
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func NewConfigFromFile(configFilePath string) (APIConfig, error) {
	var cfg APIConfig

	fmt.Printf("Loading custom config from: %s\n", configFilePath)
	customConfig, err := os.ReadFile(configFilePath)
	if err != nil {
		return cfg, fmt.Errorf("could not read config file at %s", configFilePath)
	}
	return buildConfig(customConfig)
}

func buildConfig(configJson []byte) (APIConfig, error) {
	var cfg APIConfig

	err := json.Unmarshal(configJson, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse custom config: %v", err)
	}

	// Override Config with any values explicitly set in Env Vars (PIXLISE_CONFIG_*)
	// NOTE: For []string slices, pass in a comma-separated string to the corresponding PIXLISE_CONFIG_ var
	// 			Ex: export PIXLISE_CONFIG_AdminEmails="me@example.com,you@example.com"
	reflection := reflect.ValueOf(&cfg).Elem()
	for i := 0; i < reflection.NumField(); i++ {
		fieldName := reflection.Type().Field(i).Name
		field := reflection.Field(i)
		if val, present := os.LookupEnv(fmt.Sprintf("PIXLISE_CONFIG_%s", fieldName)); present {
			// fmt.Printf("Overriding %s with env var PIXLISE_CONFIG_%s=%s", fieldName, fieldName, val)
			switch field.Kind() {
			case reflect.String:
				field.SetString(val)
			case reflect.Slice:
				if field.Type().Elem().Kind() == reflect.String {
					slicedVal := strings.Split(val, ",")
					field.Set(reflect.ValueOf(slicedVal))
				}

			case reflect.Int32:
				i, err := strconv.Atoi(val)
				if err != nil {
					fmt.Printf("Could not cast value PIXLISE_CONFIG_%s=%s to Int", fieldName, val)
					continue
				}
				field.SetInt(int64(i))
			}
		}
	}
	return cfg, nil
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
	configFilePath := flag.String("customConfigPath", "", "Path to the json file holding a set of custom config for the Pixlise API")
	flag.Parse()

	// Now that we have that, construct the Config from the possible sources
	var cfg APIConfig
	cfg.WSMaxMessageSize = 40000 // 40kb, so we can be sent a 30kb icon image+overhead. Likely needs to be larger for file uploads in time
	var err error

	// Populate API Config with contents of config.json or CUSTOM_CONFIG if supplied
	if configFilePath != nil && *configFilePath != "" {
		// Load config from a referenced json file
		cfg, err = NewConfigFromFile(*configFilePath)
	} else {
		err = errors.New("no configuration provided")
	}
	if err != nil {
		return cfg, err
	}

	if nodeCountOverride != nil && *nodeCountOverride > 0 {
		cfg.NodeCountOverride = int32(*nodeCountOverride)
	}

	if cfg.ImportJobMaxTimeSec <= 0 {
		cfg.ImportJobMaxTimeSec = uint32(10 * 60)
	}

	if cfg.PIQUANTJobMaxTimeSec <= 0 {
		cfg.PIQUANTJobMaxTimeSec = uint32(15 * 60)
	}

	cfg.KubeConfig = *kubeconfig

	return cfg, nil
}
