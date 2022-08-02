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

package msatestdata

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/core/logger"
	"github.com/pixlise/core/data-converter/importer"
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
