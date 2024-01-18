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

package dataimport

import (
	"os"
	"path/filepath"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

type DatasetCustomMeta struct {
	Title               string `json:"title"`
	DefaultContextImage string `json:"defaultContextImage"`
}

func readLocalCustomMeta(jobLog logger.ILogger, importPath string) (DatasetCustomMeta, error) {
	result := DatasetCustomMeta{}

	metapath := filepath.Join(importPath, filepaths.DatasetCustomMetaFileName)
	jobLog.Infof("Checking for custom meta: %v", metapath)

	if _, err := os.Stat(metapath); os.IsNotExist(err) {
		jobLog.Infof("Custom meta not found, ignoring...")
		return result, nil
	}

	localFS := fileaccess.FSAccess{}
	err := localFS.ReadJSON("", metapath, &result, false)
	if err != nil {
		jobLog.Errorf("Failed to read custom meta file: %v", err)
	}

	jobLog.Infof("Successfully read custom-meta")
	return result, err
}
