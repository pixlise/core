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

// Exposes various utility functions for strings, generation of valid filenames
// and random ID strings, zipping files/directories, reading/writing images
package utils

import (
	"bufio"
	"fmt"
	"os"
)

// ReadFileLines - Reads all lines in a file into a string array
func ReadFileLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func FilesEqual(aPath, bPath string) error {
	// Load the full context image from test data
	abytes, err := os.ReadFile(aPath)
	if err != nil {
		return err
	}

	bbytes, err := os.ReadFile(bPath)
	if err != nil {
		return err
	}

	if len(abytes) != len(bbytes) {
		return fmt.Errorf("%v length (%v bytes) does not match %v length (%v bytes)", aPath, len(abytes), bPath, len(bbytes))
	}

	for c := range abytes {
		if abytes[c] != bbytes[c] {
			return fmt.Errorf("%v differs from %v at idx=%v '%v'!='%v'", aPath, bPath, c, string(abytes[c]), string(bbytes[c]))
		}
	}

	return nil
}
