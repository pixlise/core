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

package jplbreadboard

import (
	"github.com/pixlise/core/v4/api/dataimport/internal/importerutils"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func processContextImages(path string, jobLog logger.ILogger, fs fileaccess.FileAccess) (map[int32]string, error) {
	jobLog.Infof("  Reading context image files from directory: %v", path)
	contextImgDirFiles, err := fs.ListObjects(path, "")
	//contextImgDirFiles, err := importerutils.GetDirListing(path, "", jobLog)

	if err != nil {
		return nil, err
	}

	return importerutils.GetContextImagesPerPMCFromListing(contextImgDirFiles, jobLog), nil
}
