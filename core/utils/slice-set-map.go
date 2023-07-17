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

import "golang.org/x/exp/constraints"

// Simple Go helper functions
// stuff that you'd expect to be part of the std lib but aren't, eg functions to search for strings
// in string arrays...

func ItemInSlice[T comparable](a T, list []T) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func AddItemsToSet[K comparable](keys []K, theSet map[K]bool) {
	for _, key := range keys {
		theSet[key] = true
	}
}

func GetMapKeys[K comparable, V any](theMap map[K]V) []K {
	result := []K{}

	for key := range theMap {
		result = append(result, key)
	}

	return result
}

func ConvertIntSlice[T constraints.Integer, F constraints.Integer](from []F) []T {
	res := make([]T, len(from))
	for i, e := range from {
		res[i] = T(e)
	}
	return res
}
