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

package quantModel

import (
	"strings"

	protos "github.com/pixlise/core/v3/generated-protos"
)

// GetWeightPercentColumnsInQuant - returns weight % columns, ones ending in _%
func GetWeightPercentColumnsInQuant(quant *protos.Quantification) []string {
	result := []string{}
	for _, label := range quant.Labels {
		if strings.HasSuffix(label, "_%") {
			result = append(result, label)
		}
	}
	return result
}

// GetQuantColumnIndex - returns index of column in quantification or -1 if not found
func GetQuantColumnIndex(quant *protos.Quantification, column string) int32 {
	for c, label := range quant.Labels {
		if label == column {
			return int32(c)
		}
	}
	return -1
}

/*
// GetQuantDetectorIndex - returns index of detector in quantification locations or -1 if not found
func GetQuantDetectorIndex(quant *protos.Quantification, detector string) int32 {
	for c, locSet := range quant.LocationSet {
		if locSet.Detector == detector {
			return int32(c)
		}
	}
	return -1
}
*/
