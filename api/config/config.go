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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/logger"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Configuration for app

// APIConfig combines env vars and config JSON values
type APIConfig struct {
	AdminEmails []string

	Auth0Domain             string
	Auth0ManagementClientID string
	Auth0ManagementSecret   string

	BuildsBucket string // Piquant download bucket
	ConfigBucket string

	CoresPerNode int32

	DataSourceSNSTopic string

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

	// Vars not set by environment
	NodeCountOverride int32
	MaxQuantNodes     int32
	KubeConfig        string // Env sets this via command line parameter
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
	customConfig, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return cfg, fmt.Errorf("could not read config file at %s", configFilePath)
	}
	return buildConfig(customConfig)
}

func NewConfigFromJsonString(customConfigStr string) (APIConfig, error) {
	customConfig := []byte(customConfigStr)
	fmt.Printf("WARNING: Passing json string via CUSTOM_CONFIG is deprecated and will soon be removed")
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
	configFilePath := flag.String("customConfigPath", "", "(optional) path to the json file holding a set of custom config for the Pixlise API")
	flag.Parse()

	// Now that we have that, construct the Config from the possible sources
	var cfg APIConfig
	var err error

	// Populate API Config with contents of config.json or CUSTOM_CONFIG if supplied
	if configFilePath != nil && *configFilePath != "" {
		// Load config from a referenced json file
		cfg, err = NewConfigFromFile(*configFilePath)
	} else {
		// Load config from a jsonString in environment variable
		customConfigStr, ok := os.LookupEnv("CUSTOM_CONFIG")
		if !ok || len(customConfigStr) <= 0 {
			return cfg, errors.New("no CUSTOM_CONFIG environment variable provided")
		} else {
			cfg, err = NewConfigFromJsonString(customConfigStr)
		}
	}
	if err != nil {
		return cfg, err
	}

	if nodeCountOverride != nil && *nodeCountOverride > 0 {
		cfg.NodeCountOverride = int32(*nodeCountOverride)
	}
	cfg.KubeConfig = *kubeconfig

	return cfg, nil
}
