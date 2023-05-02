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

// Data structures used to store expressions (both Lua and PIXLang expression code)
package expressions

import (
	"github.com/pixlise/core/v3/core/pixlUser"
)

// What users send in POST and PUT
type DataExpressionInput struct {
	Name             string            `json:"name"`
	SourceCode       string            `json:"sourceCode"`
	SourceLanguage   string            `json:"sourceLanguage"` // LUA vs PIXLANG
	Comments         string            `json:"comments"`
	Tags             []string          `json:"tags"`
	ModuleReferences []ModuleReference `json:"moduleReferences,omitempty" bson:"moduleReferences,omitempty"`
}

// Stats related to executing an expression. We get these from the UI when it runs
// an expression, and we (may) supply them when UI queries for expressions. This way the
// UI can know what expression is compatible with currently loaded quant file, and can
// "strike-through" ones that aren't. This was previously implemented client-side only
// by parsing expression text but we no longer supply ALL expression texts to client
type DataExpressionExecStats struct {
	DataRequired     []string `json:"dataRequired"`
	RuntimeMS        float32  `json:"runtimeMs"`
	TimeStampUnixSec int64    `json:"mod_unix_time_sec,omitempty"`
}

type ModuleReference struct {
	ModuleID string `json:"moduleID"`
	Version  string `json:"version"`
}

type DataExpression struct {
	ID               string                 `json:"id" bson:"_id"` // Use as Mongo ID
	Name             string                 `json:"name"`
	SourceCode       string                 `json:"sourceCode"`
	SourceLanguage   string                 `json:"sourceLanguage"` // LUA vs PIXLANG
	Comments         string                 `json:"comments"`
	Tags             []string               `json:"tags"`
	ModuleReferences []ModuleReference      `json:"moduleReferences,omitempty" bson:"moduleReferences,omitempty"`
	Origin           pixlUser.APIObjectItem `json:"origin"`
	// NOTE: if modifying below, ensure it's in sync with ExpressionDB StoreExpressionRecentRunStats()
	RecentExecStats *DataExpressionExecStats `json:"recentExecStats,omitempty" bson:"recentExecStats,omitempty"`
}

func (a DataExpression) SetTimes(userID string, t int64) {
	if a.Origin.CreatedUnixTimeSec == 0 {
		a.Origin.CreatedUnixTimeSec = t
	}
	if a.Origin.ModifiedUnixTimeSec == 0 {
		a.Origin.ModifiedUnixTimeSec = t
	}
}

type DataExpressionLookup map[string]DataExpression

// We used to store origin info in the same struct as expression...
// TODO: Remove this eventually and modify UI to work the new way too
type DataExpressionWire struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	SourceCode       string            `json:"sourceCode"`
	SourceLanguage   string            `json:"sourceLanguage"` // LUA vs PIXLANG
	Comments         string            `json:"comments"`
	Tags             []string          `json:"tags"`
	ModuleReferences []ModuleReference `json:"moduleReferences,omitempty" bson:"moduleReferences,omitempty"`
	*pixlUser.APIObjectItem
	// NOTE: if modifying below, ensure it's in sync with ExpressionDB StoreExpressionRecentRunStats()
	RecentExecStats *DataExpressionExecStats `json:"recentExecStats,omitempty" bson:"recentExecStats,omitempty"`
}
