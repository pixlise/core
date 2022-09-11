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

package msatestdata

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/data-converter/importer"
)

func processContextImages(path string, jobLog logger.ILogger) (map[int32]string, error) {
	fmt.Printf("  Reading context image files from directory: %v\n", path)
	contextImgDirFiles, err := importer.GetDirListing(path, "", jobLog)

	if err != nil {
		return nil, err
	}

	return getContextImagesPerPMCFromListing(contextImgDirFiles), nil
}

func getContextImagesPerPMCFromListing(paths []string) map[int32]string {
	result := make(map[int32]string)

	for _, pathitem := range paths {
		_, file := filepath.Split(pathitem)
		extension := filepath.Ext(file)
		if extension == ".jpg" {
			fileNameBits := strings.Split(file, "_")
			if len(fileNameBits) != 3 {
				fmt.Printf("Ignored unexpected image file name \"%v\" when searching for context images.\n", pathitem)
			} else {
				pmcStr := fileNameBits[len(fileNameBits)-1]
				pmcStr = pmcStr[0 : len(pmcStr)-len(extension)]
				pmcI, err := strconv.Atoi(pmcStr)
				if err != nil {
					fmt.Printf("Ignored unexpected image file name \"%v\", couldn't parse PMC.\n", pathitem)
				} else {
					result[int32(pmcI)] = file
				}
			}
		}
	}
	return result
}
