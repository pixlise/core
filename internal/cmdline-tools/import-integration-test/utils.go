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

package main

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v3/core/fileaccess"
)

const timeFormat = "15:04:05" // "2006-01-02 15:04:05"
var lastStartedTestName = ""

func printTestStart(name string) string {
	timeNow := time.Now().Format(timeFormat)

	fmt.Println("---------------------------------------------------------")
	fmt.Printf(" %v TEST: %v\n", timeNow, name)
	//fmt.Println("---------------------------------------------------------")

	lastStartedTestName = name

	// Not even sure why this is returned anymore, seems it's not always passed as
	// name param to printTestResult, but we use lastStartedTestName now anyway
	return name
}

var failedTestNames = []string{}

func printTestResult(err error, name string) {
	suffix := ""
	if len(name) > 0 {
		suffix = " [" + name + "]"
	}

	timeNow := time.Now().Format(timeFormat)

	if err == nil {
		fmt.Printf(" %v  PASS%v", timeNow, suffix)
	} else {
		fmt.Printf(" %v  FAILED%v: %v\n", timeNow, suffix, err)
		failedTestNames = append(failedTestNames, lastStartedTestName)
	}
	fmt.Println("")
}

func showSubHeading(heading string) {
	fmt.Println("\n---------------------------------------------------------")
	timeNow := time.Now().Format(timeFormat)
	fmt.Printf("%v %v...\n\n", timeNow, heading)
}

// Deletes with uniform logging
func deleteFile(fs fileaccess.FileAccess, bucket string, filePath string) error {
	err := fs.DeleteObject(bucket, filePath)

	if err != nil && !fs.IsNotFoundError(err) {
		err = fmt.Errorf("Failed to delete s3://%v/%v. Error: %v", bucket, filePath, err)
		//fmt.Printf(" %v\n", err)
		return err
	}

	fmt.Printf("  Deleted: s3://%v/%v\n", bucket, filePath)
	return nil
}
