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

// api - package for containing "core" API things, which are reusable
// in building any API for our platform. These should not contain
// specific PIXLISE API business logic
package api

import (
	"encoding/json"
	"net/http"

	"github.com/pixlise/core/v2/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// JSON Helper

// See:
// https://stackoverflow.com/questions/19038598/how-can-i-pretty-print-json-using-go

func ToJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Add("Content-Type", "application/json")

	if v != nil {
		enc := json.NewEncoder(w)
		enc.SetIndent("", utils.PrettyPrintIndentForJSON)
		enc.Encode(v)
	}
}
