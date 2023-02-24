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

package dataExpression

import (
	"fmt"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/utils"
)

type expressionType string

const (
	contextImage expressionType = "ContextImage"
	binaryPlot                  = "BinaryPlot"
	ternaryPlot                 = "TernaryPlot"
	chordDiagram                = "ChordDiagram"
	all                         = "All"
)

func (et expressionType) IsValid() error {
	switch et {
	case contextImage, binaryPlot, ternaryPlot, chordDiagram, all:
		return nil
	}
	return fmt.Errorf("Invalid expression type: %v", et)
}

// DataExpressionInput - only public so we can use it embedded in dataExpression
type DataExpressionInput struct {
	Name       string         `json:"name"`
	Expression string         `json:"expression"`
	Type       expressionType `json:"type"`
	Comments   string         `json:"comments"`
	Tags       []string       `json:"tags"`
}

// Stats related to executing an expression. We get these from the UI when it runs
// an expression, and we (may) supply them when UI queries for expressions. This way the
// UI can know what expression is compatible with currently loaded quant file, and can
// "strike-through" ones that aren't. This was previously implemented client-side only
// by parsing expression text but we no longer supply ALL expression texts to client
type DataExpressionExecStats struct {
	DataRequired     []string `json:"dataRequired"`
	RuntimeMS        int32    `json:"runtimeMs"`
	TimeStampUnixSec int64    `json:"mod_unix_time_sec,omitempty"`
}

type DataExpression struct {
	*DataExpressionInput
	*pixlUser.APIObjectItem
}

// When sending out, we include more data than we store each item with
type DataExpressionWire struct {
	*DataExpression
	RecentExecStats *DataExpressionExecStats `json:"recentExecStats,omitempty"`
}

func (a DataExpression) SetTimes(userID string, t int64) {
	if a.CreatedUnixTimeSec == 0 {
		a.CreatedUnixTimeSec = t
	}
	if a.ModifiedUnixTimeSec == 0 {
		a.ModifiedUnixTimeSec = t
	}
}

type DataExpressionLookup map[string]DataExpression

func ReadExpressionData(svcs *services.APIServices, s3Path string) (DataExpressionLookup, error) {
	itemLookup := DataExpressionLookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)
	return itemLookup, err
}

func GetListing(svcs *services.APIServices, userID string, outMap *DataExpressionLookup) error {
	s3PathFrom := filepaths.GetExpressionPath(userID)
	sharedFile := userID == pixlUser.ShareUserID

	items, err := ReadExpressionData(svcs, s3PathFrom)
	if err != nil {
		return err
	}

	for id, item := range items {
		// We modify the ids of shared items, so if passed to GET/PUT/DELETE we know this refers to something that's shared
		saveID := id
		if sharedFile {
			saveID = utils.SharedItemIDPrefix + id
		}
		item.Shared = sharedFile
		(*outMap)[saveID] = item
	}

	return nil
}
