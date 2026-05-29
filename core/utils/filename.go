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
	"fmt"
	"strings"
)

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

func ApplyIndexToFileName(name string, index uint, applyIndex bool) string {
	if !applyIndex {
		return name
	}

	// Ideally we could use:
	//ext := filepath.Ext(name)
	// But this is not the same behaviour as in PIQUANT when it outputs
	// a file name, and instead of modifying PIQUANT we can just change
	// how we generate file names here
	ext := ""
	pos := strings.Index(name, ".")
	if pos >= 0 {
		ext = name[pos:]
	}
	return fmt.Sprintf("%v%06d%v", name[0:len(name)-len(ext)], index, ext)
}
