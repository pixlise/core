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

package quantification

import (
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/quantification/quantRunner"
	"github.com/pixlise/core/v4/api/services"
)

func savePMCList(svcs *services.APIServices, jobBucket string, contents string, nodeNumber int, jobDataPath string) (string, error) {
	pmcListName := quantRunner.MakePMCFileName(nodeNumber)
	savePath := path.Join(jobDataPath, pmcListName)

	err := svcs.FS.WriteObject(jobBucket, savePath, []byte(contents))
	if err != nil {
		// Couldn't save it, no point continuing, we don't want a quantification with a section missing!
		return pmcListName, fmt.Errorf("error when writing node PMC list: %v. Error: %v", savePath, err)
	}

	return pmcListName, nil
}
