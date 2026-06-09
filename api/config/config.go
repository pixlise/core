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

	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Configuration for app

// APIConfig combines env vars and config JSON values
type APIConfig struct {
	AdminEmails []string

	// Auth0 settings
	Auth0Domain             string
	Auth0ManagementClientID string
	Auth0ManagementSecret   string
	Auth0ClientSecret       string
	Auth0Namespace          string

	// New user creation
	Auth0NewUserRoleID string
	DefaultUserGroupId string // The GroupId of the group a new user is added to by default as a member

	// Buckets
	ConfigBucket       string
	PiquantJobsBucket  string // PIQUANT job scratch drive
	DatasetsBucket     string
	UsersBucket        string
	ManualUploadBucket string
	DataBackupBucket   string

	DataSourceSNSTopic string

	EnvironmentName string

	// Logging/monitoring of PIXLISE
	LogLevel       logger.LogLevel // Can be changed at runtime, but if API restarts, it goes back to configured value
	SentryEndpoint string

	// Mongo Connection
	MongoSecret string
	MongoDebug  bool

	// Zenodo config
	ZenodoURI         string
	ZenodoAccessToken string

	// Web Socket config
	WSWriteWaitMs       uint
	WSPongWaitMs        uint
	WSPingPeriodMs      uint
	WSMaxMessageSize    uint
	WSMessageBufferSize uint

	// Local file caching (from S3 to where API is running)
	MaxFileCacheAgeSec    uint
	MaxFileCacheSizeBytes uint

	// Max time we allow memoised item to exist in DB and not be retrieved.
	// If it hasn't been accessed in this many seconds, consider it stale & delete it!
	MaxUnretrievedMemoisationAgeSec uint

	// How often we run memoisation GC
	MemoisationGCIntervalSec uint

	// Admin-only features: backup & restore settings, and allowing impersonate user menu option
	BackupEnabled             bool
	RestoreEnabled            bool
	RestoreExcludeCollections []string
	ImpersonateEnabled        bool

	// Settings that control what kind of job processing EC2 instances we create
	Jobs JobConfig

	// Deprecated configs, these should all disappear when we remove the old job code
	ImportJobMaxTimeSec uint32
	KubeConfig          string // Env sets this via command line parameter
	QuantExecutor       string
	QuantNamespace      string // Used for running large multi-node quants
	HotQuantNamespace   string // Used for faster PIQUANT runs, eg executing a spectral fit
	KubernetesLocation  string // "internal" vs "external"
}

// JobConfig contains all configs required to be able to run jobs by the back-end. This can involve starting quants or other jobs
// locally in Docker, or starting up a JobNode (an EC2 instance with a pixlise-job-node executable running) to execute the job
type JobConfig struct {
	LegacyJobs bool

	// Configuring AWS EC2 instance type for job node
	CoresPerNode  uint
	InstanceType  string
	AMI           string
	KeyName       string
	SecurityGroup string

	// How many nodes to run, and limits for how long they run
	MaxNodeRunTimeSec uint32
	NodeCountOverride uint // Forces PMC list generation to create this many nodes. Mainly usable for testing.
	MaxQuantNodes     uint // Limiting how many nodes we run simultaneously

	// Jobs run in docker, these configure what container and settings they use
	AWSSecret         string
	NodeS3Path        string
	RunnerDockerImage string // Docker image to run expressions, python code and PIQUANT on the back end
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
var argNodeCountOverride int

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

	if nodeCountOverride != nil {
		argNodeCountOverride = *nodeCountOverride
	}

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

	// If our job structure is empty, we attempt to read a job config from the config
	// bucket
	if len(cfg.Jobs.RunnerDockerImage) <= 0 {
		cfg.Jobs = JobConfig{}
	}

	cfg.KubeConfig = *kubeconfig

	applyConfigLimits(&cfg)

	return cfg, nil
}

func applyConfigLimits(cfg *APIConfig) {
	if cfg.Jobs.CoresPerNode <= 0 {
		cfg.Jobs.CoresPerNode = 6 // Core count can't be 0!
	}

	if cfg.Jobs.MaxQuantNodes <= 0 {
		cfg.Jobs.MaxQuantNodes = 120
	}

	if cfg.Jobs.MaxNodeRunTimeSec <= 0 {
		cfg.Jobs.MaxNodeRunTimeSec = 30 * 60
	}

	if argNodeCountOverride > 0 {
		cfg.Jobs.NodeCountOverride = uint(argNodeCountOverride)
	}

	if cfg.ImportJobMaxTimeSec <= 0 {
		cfg.ImportJobMaxTimeSec = uint32(10 * 60)
	}

	if cfg.MaxUnretrievedMemoisationAgeSec <= 0 {
		cfg.MaxUnretrievedMemoisationAgeSec = 86400 * 30
	}

	if cfg.MemoisationGCIntervalSec <= 0 {
		cfg.MemoisationGCIntervalSec = 3600
	}
}

func ReadJobConfig(cfg *APIConfig, fs fileaccess.S3Access) error {
	if err := fs.ReadJSON(cfg.ConfigBucket, "job-config.json", &cfg.Jobs, false); err != nil {
		return err
	}

	applyConfigLimits(cfg)
	return nil
}
