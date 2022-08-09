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

package importer

import (
	"fmt"

	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/data-converter/converterModels"
)

type Importer interface {
	Import(importPath string, pseudoIntensityRangesPath string, jobLog logger.ILogger) (*converterModels.OutputData, string, error)
}

func LogIfMoreFoundMSA(m converterModels.DetectorSampleByPMC, typename string, morethan int) {
	for k, v := range m {
		if len(v) > morethan {
			fmt.Printf("PMC %d has %d %s entries\n", k, len(v), typename)
		}
	}
}
