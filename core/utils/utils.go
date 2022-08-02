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

package utils

import (
	"fmt"
	"io/ioutil"
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
