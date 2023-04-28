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

// Defines handlers, kind of like base classes for endpoints
package handlers

import (
	"path"
	"strings"
)

// Some constants
const downloadCacheMaxAgeSec = 604800 // how long we tell browser to cache files for, in sec
const downloadCacheMinMaxAgeSec = 120

// Public general-purpose functions
func MakeEndpointPath(pathPrefix string, pathParamNames ...string) string {
	vals := []string{"/" + pathPrefix}

	for _, param := range pathParamNames {
		vals = append(vals, "{"+strings.Trim(param, "/")+"}")
	}

	return path.Join(vals...)
}

// The rest can be found in use-specific handler go files
