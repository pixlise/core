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

package modules

import (
	"fmt"
	"testing"
)

func Test_IsValidModuleName(t *testing.T) {
	expResult := []bool{
		true,
		false,
		true,
		false,
		true,
		false,
		false,
		false,
		false,
		false,
		true,
	}
	names := []string{
		"Hello",
		"Hello World",
		"Var1",
		"1Var",
		"_1Var",
		"WeirdChar$",
		"Weird.Char",
		"Weird[Char]",
		"_",
		"__",
		"__a",
		// Also should probably guard against reserved function names, but we can let the UI do this check
		// because only it knows what the function names are!
	}

	for c := 0; c < len(names); c++ {
		if IsValidModuleName(names[c]) != expResult[c] {
			t.Errorf("Expected %v to return valid=%v", names[c], expResult[c])
		}
	}
}

func Example_SemVer() {
	fmt.Println(SemanticVersionToString(SemanticVersion{Major: 10, Minor: 13, Patch: 14}))
	v, err := SemanticVersionFromString("11.13.15")
	fmt.Printf("%v|%v\n", v, err)
	v, err = SemanticVersionFromString("11.13.15.16")
	fmt.Printf("%v|%v\n", v, err)
	v, err = SemanticVersionFromString("11.Hello.16")
	fmt.Printf("%v|%v\n", v, err)

	// Output:
	// 10.13.14
	// {11 13 15}|<nil>
	// {0 0 0}|Invalid semantic version: 11.13.15.16
	// {0 0 0}|Failed to parse version 11.Hello.16, part Hello is not a number
}
