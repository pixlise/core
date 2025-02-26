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
	"os"
	"testing"
)

// Check the summary filename gets created correctly
func Test_InitializeConfigWithFile(t *testing.T) {
	want := "datasetsBucket"
	cfg, err := NewConfigFromFile("./test-data/example_config.json")
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	if cfg.DatasetsBucket != want {
		t.Errorf("cfg.DatasetsBucket got %q; want: %q", cfg.DatasetsBucket, want)
	}
}

// Check that the config can be overridden with Environment Variables
func Test_OverrideConfigWithEnvVars(t *testing.T) {
	want := "ENV-SET-DatasetsBucket"
	os.Setenv("PIXLISE_CONFIG_DatasetsBucket", want)
	cfg, err := NewConfigFromFile("./test-data/example_config.json")
	if err != nil {
		t.Fatalf("Error initializing config: %v", err)
	}
	if cfg.DatasetsBucket != want {
		t.Errorf("cfg.DatasetsBucket got %q; want: %q", cfg.DatasetsBucket, want)
	}
}
