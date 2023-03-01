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
	"regexp"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/pixlUser"
)

// DataModuleInput - This defines a "module" of code that can be used as part of expressions. At the time
// of writing (and hopefully indefinitely), Lua is the programming language PIXLISE will support. The text
// contained in the "module" field is to be a valid Lua module, but the outer "wrapping" around it is added
// at runtime.

// As an example:
// --- Added automatically at runtime to a module called MyModule ---
// MyModule = {}
// --- End automatically added section ---
//
// function MyModule.add(a, b)
//     return a+b
// end
//
// --- Added automatically at runtime to a module called MyModule ---
// return MyModule
// --- End automatically added section ---

// Modules similarities with Expressions:
// - Both have a unique string ID, an editable name, and comments
// - Both store text with executable code in it, and listing both returns all data except the text (because it's large)
// - Both can be queried by ID to get the full object (including text)

// Modules differences from Expressions:
// - Module names must be valid Lua variable names, because they are imported into Lua and used as a variable
// - Modules cannot be deleted, only new versions can be created (using PUT)
// - Modules store all previous versions, and each of these can be queried (GET) to get the module code
// - When listing Modules, you get the metadata for a module and a list of valid version numbers, along with associated tags
//   for each version number

// What users send in POST
type DataModuleInput struct {
	Name       string   `json:"name"`       // Editable name
	SourceCode string   `json:"sourceCode"` // The module executable code
	Comments   string   `json:"comments"`   // Editable comments
	Tags       []string `json:"tags"`       // Any tags for this version
}

// And what we get in PUT for new versions being uploaded
type DataModuleVersionInput struct {
	SourceCode string   `json:"sourceCode"` // The module executable code
	Comments   string   `json:"comments"`   // Editable comments
	Tags       []string `json:"tags"`       // Any tags for this version
}

type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

func SemanticVersionToString(v SemanticVersion) string {
	return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Patch)
}

func SemanticVersionFromString(v string) (SemanticVersion, error) {
	result := SemanticVersion{}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("Invalid semantic version: %v", v)
	}
	nums := []int{}
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return result, fmt.Errorf("Failed to parse version %v, part %v is not a number", v, part)
		}
		nums = append(nums, num)
	}

	result.Major = nums[0]
	result.Minor = nums[1]
	result.Patch = nums[2]

	return result, nil
}

// Stored version of a module
type DataModuleVersion struct {
	ModuleID         string          `json:"moduleID"` // The ID of the module we belong to
	SourceCode       string          `json:"sourceCode"`
	Version          SemanticVersion `json:"version"`
	Tags             []string        `json:"tags"`
	Comments         string          `json:"comments"`
	TimeStampUnixSec int64           `json:"mod_unix_time_sec"`
}

// Stored module object itself
type DataModule struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Comments string                 `json:"comments"`
	Origin   pixlUser.APIObjectItem `json:"origin"`
}

// What we send out to users - notice versions only contains version numbers & tags
type DataModuleVersionWire struct {
	Version          string   `json:"version"`
	Tags             []string `json:"tags"`
	Comments         string   `json:"comments"`
	TimeStampUnixSec int64    `json:"mod_unix_time_sec"`
}

// As above, but with source field
type DataModuleVersionSourceWire struct {
	SourceCode string `json:"sourceCode"`
	*DataModuleVersionWire
}

type DataModuleWire struct {
	*DataModule
	Versions []DataModuleVersionWire `json:"versions"`
}

// And what we send for a specific module version request
type DataModuleSpecificVersionWire struct {
	*DataModule
	Version DataModuleVersionSourceWire `json:"version"`
}

type DataModuleWireLookup map[string]DataModuleWire

// Some validation functions
func IsValidModuleName(name string) bool {
	// Names must be valid Lua variable names...
	match, err := regexp.MatchString("^[A-Za-z]$|^[A-Za-z_]+[A-Za-z0-9_]*[A-Za-z0-9]$", name)
	if err != nil {
		return false
	}
	return match
}
