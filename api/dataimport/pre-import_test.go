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

package dataimport

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"

	"github.com/pixlise/core/v4/api/dataimport/sdfToRSI"
	"github.com/pixlise/core/v4/core/logger"
)

// Trigger for a manual dataset regeneration (user clicks save button on dataset edit page)
func Example_ProcessEM() {
	p := "./test-data/PreImport/20240805_EM_V8.2_ATP_Test_65_A_Day_in_the_Life.zip"
	zFile, err := os.ReadFile(p)

	if err == nil {
		var z *zip.Reader
		z, err = zip.NewReader(bytes.NewReader(zFile), int64(len(zFile)))

		if err == nil {
			l := logger.StdOutLoggerForTest{}

			localTemp, sdfLocalPath, _, _, err := startEMProcess("123", z, zFile, &l)
			fmt.Printf("startEMProcess err=%v\n", err)
			genFiles, rtts, err := sdfToRSI.ConvertSDFtoRSIs(sdfLocalPath, localTemp)
			fmt.Printf("genFiles: %v\nrtts: %v\nerr: %v\n", genFiles, rtts, err)
		}
	}

	// Output:
	// startEMProcess err=<nil>
	// genFiles: [RSI-453.csv HK-453.csv]
	// rtts: [453]
	// err: <nil>
}
