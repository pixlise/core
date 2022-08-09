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

package fileaccess

import "strings"

// Generic interface for reading/writing files asynchronously
// We could have used OS level things but we want to be able to
// swap this out with AWS S3, other cloud APIs or local file system
// so we now have this interface to code against, and can implement
// it any way we like.

// Besides just needing a path, we may need a drive or bucket or account
// id at the start of a path.

type FileAccess interface {
	ListObjects(bucket string, prefix string) ([]string, error)

	ReadObject(bucket string, path string) ([]byte, error)
	WriteObject(bucket string, path string, data []byte) error

	ReadJSON(bucket string, s3Path string, itemsPtr interface{}, emptyIfNotFound bool) error
	WriteJSON(bucket string, s3Path string, itemsPtr interface{}) error
	// A few places in the code need to write non-pretty-printed JSON for those
	// files to work with Athena queries. Instead of adding a flag to WriteJSON
	// this is easier to implement. Searching for WriteJSON still returns these!
	WriteJSONNoIndent(bucket string, s3Path string, itemsPtr interface{}) error

	DeleteObject(bucket string, path string) error

	CopyObject(srcBucket string, srcPath string, dstBucket string, dstPath string) error

	EmptyObjects(targetBucket string) error

	IsNotFoundError(err error) bool
}

func MakeValidObjectName(name string) string {
	//name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "$", "")
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ReplaceAll(name, "!", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	//name = strings.ReplaceAll(name, "-", "_")

	return name
}

// Is this string a valid name to use as an AWS object name?
func IsValidObjectName(name string) bool {
	// Names should be non-zero length containing some non-crazy characters
	if len(name) <= 0 {
		return false
	}

	if strings.ContainsAny(name, "\"") {
		return false
	}

	return true
}
