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
	"fmt"
	"os"
	"testing"
)

// Check the summary filename gets created correctly
func Test_InitializeConfigWithFile(t *testing.T) {
	var cfg APIConfig
	want := "buildsBucket"
	cfg, err := NewConfigFromFile("./example_config.json", cfg)
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	if cfg.BuildsBucket != want {
		t.Errorf("cfg.BuildsBucket got %q; want: %q", cfg.BuildsBucket, want)
	}
}

// Check the quant path is calculated correctly
func Test_InitializeConfigWithJsonString(t *testing.T) {
	var cfg APIConfig
	want := "buildsBucketCustomConfig"
	configStr := fmt.Sprintf(`{"BuildsBucket": "%s"}`, want)
	cfg, err := NewConfigFromJsonString(configStr, cfg)
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	if cfg.BuildsBucket != want {
		t.Errorf("cfg.BuildsBucket got %q; want: %q", cfg.BuildsBucket, want)
	}
}

// Check that the config can be overridden with Environment Variables
func Test_OverrideConfigWithEnvVars(t *testing.T) {
	var cfg APIConfig
	want := "ENV-SET-BuildsBucket"
	os.Setenv("PIXLISE_CONFIG_BuildsBucket", want)
	cfg, err := NewConfigFromFile("./example_config.json", cfg)
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	if cfg.BuildsBucket != want {
		t.Errorf("cfg.BuildsBucket got %q; want: %q", cfg.BuildsBucket, want)
	}

}
