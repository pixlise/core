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

// Provides a higher-level file access interface which is implemented using local file storage as well as AWS S3. This makes writing code
// that is agnostic to file storage medium much easier. Both are tested with the same unit testing framework to ensure they are compatible
// NOTE: In the FileAccess interface any reference to a bucket when used with the local file storage model is concatented with
// the path, so it could be thought of as a reference to the disk the path is on, or the root of where relative paths are
// available from
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
	// Effectively recursive listing of files in a given directory
	ListObjects(bucket string, prefix string) ([]string, error)

	// Does a file at a given path exist
	ObjectExists(rootPath string, path string) (bool, error)

	// Reads a file as bytes
	ReadObject(bucket string, path string) ([]byte, error)
	// Writes a file as bytes
	WriteObject(bucket string, path string, data []byte) error

	// Reads a file as JSON and decodes it into itemsPtr
	ReadJSON(bucket string, s3Path string, itemsPtr interface{}, emptyIfNotFound bool) error
	// Writes itemsPtr as a JSON file
	WriteJSON(bucket string, s3Path string, itemsPtr interface{}) error

	// Same as WriteJSON, but a few places in the code need to write
	// non-pretty-printed JSON for those files to work with Athena queries for
	// example. Instead of adding a flag to WriteJSON this is easier to implement.
	// Searching for WriteJSON still returns these!
	WriteJSONNoIndent(bucket string, s3Path string, itemsPtr interface{}) error

	// Delete a file
	DeleteObject(bucket string, path string) error

	// Copy a file
	CopyObject(srcBucket string, srcPath string, dstBucket string, dstPath string) error

	// Effectively performs "rm -rf" of all files the given bucket/root directory
	EmptyObjects(targetBucket string) error

	// Checks if the given error is a "not found" error for the implementation. This is because
	// AWS S3 would provide a different "not found" error than would a local file system fopen() failing
	IsNotFoundError(err error) bool
}

// Turns a string name potentially typed by a user into a file name that should be valid for storage
// in anything we store in. This removes or replaces illegal characters with _.
func MakeValidObjectName(name string, allowSpace bool) string {
	//name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "$", "")
	name = strings.ReplaceAll(name, "#", "")
	name = strings.ReplaceAll(name, "!", "")
	name = strings.ReplaceAll(name, "'", "")
	name = strings.ReplaceAll(name, ";", "")
	name = strings.ReplaceAll(name, "&", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	//name = strings.ReplaceAll(name, "-", "_")
	if !allowSpace {
		name = strings.ReplaceAll(name, " ", "_")
	}

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
