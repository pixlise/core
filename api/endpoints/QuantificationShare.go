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

package endpoints

import (
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/core/quantModel"
)

func quantificationShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying ID of quantification to share. If it's valid, we need to copy that quant
	// and it's summary file into the shared user area, thereby implementing "share a copy"
	jobID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	err := quantModel.ShareQuantification(params.Svcs, params.UserInfo.UserID, datasetID, jobID)
	if err != nil {
		return nil, err
	}

	return "shared", nil
}
