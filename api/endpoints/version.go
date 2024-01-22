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
	"strings"

	"github.com/pixlise/core/v4/api/piquant"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Getting component versions

func getAPIVersion() string {
	ver := services.ApiVersion
	if len(services.ApiVersion) <= 0 {
		ver = "(Local build)"
	}

	if len(services.GitHash) > 0 {
		hashEnd := 8
		if len(services.GitHash) < 8 {
			hashEnd = len(services.GitHash)
		}
		ver += "-" + services.GitHash[0:hashEnd]
	}

	return ver
}

func getVersion(svcs *services.APIServices) *protos.VersionResponse {
	result := &protos.VersionResponse{}
	result.Versions = []*protos.VersionResponse_Version{
		{
			Component: "API",
			Version:   getAPIVersion(),
		},
	}

	if svcs.MongoDB != nil { // When running tests, this will be nil
		piquantVersion, err := piquant.GetPiquantVersion(svcs)
		piquantVerString := "error"
		if err == nil {
			piquantVerString = piquantVersion.Version

			// If we can, just get the end
			parts := strings.Split(piquantVerString, "/")
			if len(parts) > 0 {
				piquantVerString = parts[len(parts)-1]
			}
		}

		result.Versions = append(result.Versions, &protos.VersionResponse_Version{
			Component: "PIQUANT",
			Version:   piquantVerString,
		})
	}

	return result
}

func GetVersionProtobuf(params apiRouter.ApiHandlerGenericPublicParams) error {
	result := getVersion(params.Svcs)
	utils.SendProtoBinary(params.Writer, result)
	return nil
}

func GetVersionJSON(params apiRouter.ApiHandlerGenericPublicParams) error {
	result := getVersion(params.Svcs)
	utils.SendProtoJSON(params.Writer, result)
	return nil
}
