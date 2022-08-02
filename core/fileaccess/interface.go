// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

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
