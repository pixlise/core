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

package dataExpression

import (
	"fmt"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/utils"
)

const expressionFile = "DataExpressions.json"

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
}

type DataExpression struct {
	*DataExpressionInput
	*pixlUser.APIObjectItem
}

type DataExpressionLookup map[string]DataExpression

func ReadExpressionData(svcs *services.APIServices, s3Path string) (DataExpressionLookup, error) {
	itemLookup := DataExpressionLookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)
	return itemLookup, err
}

func GetListing(svcs *services.APIServices, userID string, outMap *DataExpressionLookup) error {
	s3PathFrom := filepaths.GetUserContentPath(userID, expressionFile)
	sharedFile := userID == pixlUser.ShareUserID

	items, err := ReadExpressionData(svcs, s3PathFrom)
	if err != nil {
		return err
	}

	for id, item := range items {
		// We modify the ids of shared items, so if passed to GET/PUT/DELETE we know this refers to something that's
		saveID := id
		if sharedFile {
			saveID = utils.SharedItemIDPrefix + id
		}
		item.Shared = sharedFile
		(*outMap)[saveID] = item
	}

	return nil
}
