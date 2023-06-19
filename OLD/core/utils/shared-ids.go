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

package utils

// Any item that we share will have its ID prefixed with shared-. Not necessarily as stored in S3
// because it may sit in a known shared directory, but when the API sends out a shared object, it
// must be prefixed this way
const SharedItemIDPrefix = "shared-"

// StripSharedItemIDPrefix - Strips shared prefix and returns true if object was shared
func StripSharedItemIDPrefix(ID string) (string, bool) {
	prefixLen := len(SharedItemIDPrefix)
	if len(ID) > prefixLen && ID[0:prefixLen] == SharedItemIDPrefix {
		return ID[prefixLen:], true
	}
	return ID, false
}
