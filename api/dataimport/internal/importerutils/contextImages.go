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

package importerutils

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/core/logger"
)

// Check what the minimum PMC is we have a context image for
func GetMinimumContextPMC(contextImgsPerPMC map[int32]string) int32 {
	minContextPMC := int32(0)

	for contextPMC := range contextImgsPerPMC {
		if minContextPMC == 0 || contextPMC < minContextPMC {
			minContextPMC = contextPMC
		}
	}
	if minContextPMC == 0 {
		minContextPMC = 1
	}

	return minContextPMC
}

func GetContextImagesPerPMCFromListing(paths []string, jobLog logger.ILogger) map[int32]string {
	result := make(map[int32]string)

	for _, pathitem := range paths {
		_, file := filepath.Split(pathitem)
		extension := filepath.Ext(file)
		if extension == ".jpg" {
			fileNameBits := strings.Split(file, "_")
			if len(fileNameBits) != 3 {
				jobLog.Infof("Ignored unexpected image file name \"%v\" when searching for context images.", pathitem)
			} else {
				pmcStr := fileNameBits[len(fileNameBits)-1]
				pmcStr = pmcStr[0 : len(pmcStr)-len(extension)]
				pmcI, err := strconv.Atoi(pmcStr)
				if err != nil {
					jobLog.Infof("Ignored unexpected image file name \"%v\", couldn't parse PMC.", pathitem)
				} else {
					result[int32(pmcI)] = file
				}
			}
		}
	}
	return result
}
