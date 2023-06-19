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

package importtime

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
)

const tempImportJSON = "./temp-test-data/DatasetConfig/import-times.json"

func Example_GetDatasetImportUnixTimeSec() {
	defer os.RemoveAll("./temp-test-data")

	fmt.Printf("Setup: %v\n", os.MkdirAll("./temp-test-data/DatasetConfig", 0777))

	fs := fileaccess.FSAccess{}

	// Missing file
	ts, err := GetDatasetImportUnixTimeSec(&fs, "./not-exist/", "ds-123")
	fmt.Printf("%v, %v\n", ts, err)

	// JSON file is bad
	fmt.Printf("Test data: %v\n", ioutil.WriteFile(tempImportJSON, []byte("{\"times\": {\"ds-123\": \"hello\"}}"), 0777))

	ts, err = GetDatasetImportUnixTimeSec(&fs, "./temp-test-data", "ds-123")
	fmt.Printf("%v, %v\n", ts, err)

	// Dataset in file
	fmt.Printf("Test data: %v\n", ioutil.WriteFile(tempImportJSON, []byte("{\"times\": {\"ds-123\": 1234567890}}"), 0777))
	ts, err = GetDatasetImportUnixTimeSec(&fs, "./temp-test-data", "ds-123")
	fmt.Printf("%v, %v\n", ts, err)

	// Dataset not in file
	ts, err = GetDatasetImportUnixTimeSec(&fs, "./temp-test-data", "ds-not-exist")
	fmt.Printf("%v, %v\n", ts, err)

	// Output:
	// Setup: <nil>
	// 0, <nil>
	// Test data: <nil>
	// 0, json: cannot unmarshal string into Go struct field LastImportTimes.times of type int
	// Test data: <nil>
	// 1234567890, <nil>
	// 0, <nil>
}

func Example_SaveDatasetImportUnixTimeSec() {
	defer os.RemoveAll("./temp-test-data")

	fmt.Printf("Setup: %v\n", os.MkdirAll("./temp-test-data/DatasetConfig", 0777))

	fs := fileaccess.FSAccess{}
	log := logger.NullLogger{}

	// No JSON file
	err := SaveDatasetImportUnixTimeSec(&fs, &log, "./temp-test-data/", "ds-123", 1234567890)
	file, err2 := ioutil.ReadFile(tempImportJSON)
	fmt.Printf("%v, %v, %v\n", err, err2, string(file))

	// Overwrite with new timestamp
	err = SaveDatasetImportUnixTimeSec(&fs, &log, "./temp-test-data/", "ds-123", 2222222222)
	file, err2 = ioutil.ReadFile(tempImportJSON)
	fmt.Printf("%v, %v, %v\n", err, err2, string(file))

	// New dataset ID
	err = SaveDatasetImportUnixTimeSec(&fs, &log, "./temp-test-data/", "ds-444", 4444444444)
	file, err2 = ioutil.ReadFile(tempImportJSON)
	fmt.Printf("%v, %v, %v\n", err, err2, string(file))

	// JSON file is bad
	fmt.Printf("Test data: %v\n", ioutil.WriteFile(tempImportJSON, []byte("{\"times\": {\"ds-123\": \"hello\"}}"), 0777))

	err = SaveDatasetImportUnixTimeSec(&fs, &log, "./temp-test-data/", "ds-123", 5555555555)
	file, err2 = ioutil.ReadFile(tempImportJSON)
	fmt.Printf("%v, %v, %v\n", err, err2, string(file))

	// Output:
	// Setup: <nil>
	// <nil>, <nil>, {"times":{"ds-123":1234567890}}
	// <nil>, <nil>, {"times":{"ds-123":2222222222}}
	// <nil>, <nil>, {"times":{"ds-123":2222222222,"ds-444":4444444444}}
	// Test data: <nil>
	// <nil>, <nil>, {"times":{"ds-123":5555555555}}
}
