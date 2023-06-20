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
