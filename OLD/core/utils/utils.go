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
	"io/ioutil"
	"os"
)

// Simple Go helper functions
// stuff that you'd expect to be part of the std lib but aren't, eg functions to search for strings
// in string arrays...

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringSlicesEqual(test []string, ans []string) bool {
	if len(test) != len(ans) {
		return false
	}

	for c := range test {
		if test[c] != ans[c] {
			return false
		}
	}

	return true
}

// See comments about making this generic... search for REFACTOR, TODO or utils.SetStringsInMap()
func SetStringsInMap(vals []string, theMap map[string]bool) {
	for _, val := range vals {
		theMap[val] = true
	}
}

// REFACTOR: TODO: Make this more generic... and/or make an int version
// FAIL... this seems to not be compatible with ANYTHING??? func GetStringMapKeys(theMap map[string]interface{}) []string {
func GetStringMapKeys(theMap map[string]bool) []string {
	result := []string{}

	for key := range theMap {
		result = append(result, key)
	}

	return result
}

func ReplaceStringsInSlice(vals []string, replacements map[string]string) {
	for idx, val := range vals {
		if replacement, ok := replacements[val]; ok {
			vals[idx] = replacement
		}
	}
}

func AbsI64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func FilesEqual(aPath, bPath string) error {
	// Load the full context image from test data
	abytes, err := ioutil.ReadFile(aPath)
	if err != nil {
		return err
	}

	bbytes, err := ioutil.ReadFile(bPath)
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

// MakeSaveableFileName - Given a name which may not be acceptable as a file name, generate a string for a file name
// that won't have issues. This replaces bad characters like slashes with spaces, etc
func MakeSaveableFileName(name string) string {
	result := ""
	for _, ch := range name {
		if ch >= 'a' && ch <= 'z' ||
			ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' ||
			ch == ' ' ||
			ch == '-' ||
			ch == '_' ||
			ch == '%' ||
			ch == '\'' ||
			ch == '"' ||
			ch == '(' ||
			ch == ')' ||
			ch == '.' ||
			ch == ',' {
			result += string(ch)
		} else {
			result += " "
		}
	}

	return result
}

// PrettyPrintIndentForJSON Pretty-print indenting of JSON
const PrettyPrintIndentForJSON = "    "

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
