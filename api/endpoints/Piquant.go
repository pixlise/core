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

package endpoints

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/detector"
	"github.com/pixlise/core/core/piquant"
)

type quantConfig struct {
	PixliseConfig detector.DetectorConfig `json:"pixliseConfig"`
	QuantConfig   piquant.PiquantConfig   `json:"quantConfig"`
}

type detectorConfigListing struct {
	ConfigNames []string `json:"configNames"`
}

// Downloading PIQUANT
type piquantDownloadHandler struct {
	svcs *services.APIServices
}

type piquantDownloadable struct {
	BuildVersion  string `json:"buildVersion"`
	BuildDate     int64  `json:"buildDateUnixSec"`
	FileName      string `json:"fileName"`
	FileSizeBytes int64  `json:"fileSizeBytes"`
	DownloadLink  string `json:"downloadUrl"`
	OS            string `json:"os"`
}

type piquantDownloadListing struct {
	DownloadItems []piquantDownloadable `json:"downloadItems"`
}

const versionIdentifier = "version"

func registerPiquantHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "piquant"
	const configPrefix = "config"
	const versionPath = "version"
	const downloadPrefix = "download"

	// Gets all config names
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+configPrefix), apiRouter.MakeMethodPermission("GET", permission.PermWritePiquantConfig), piquantConfigList)

	// Gets config versions (for a given name)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+configPrefix, idIdentifier, "versions"), apiRouter.MakeMethodPermission("GET", permission.PermWritePiquantConfig), piquantConfigVersionsList)

	// Gets a config (for given name+version)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+configPrefix, idIdentifier, "version", versionIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermWritePiquantConfig), piquantConfigGet)

	// Listing PIQUANT builds that can be downloaded
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+downloadPrefix), apiRouter.MakeMethodPermission("GET", permission.PermDownloadPiquant), piquantDownloadList)

	// Setting/getting PIQUANT version string
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+versionPath), apiRouter.MakeMethodPermission("GET", permission.PermWritePiquantConfig), piquantVersionGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/"+versionPath), apiRouter.MakeMethodPermission("POST", permission.PermWritePiquantConfig), piquantVersionPost)
}

func piquantConfigList(params handlers.ApiHandlerParams) (interface{}, error) {
	// Return a list of all piquant configs we have stored
	// TODO: Handle paging... this could eventually be > 1000 files, but that's a while away!
	paths, err := params.Svcs.FS.ListObjects(params.Svcs.Config.ConfigBucket, filepaths.RootDetectorConfig+"/")
	if err != nil {
		params.Svcs.Log.Errorf("Failed to list piquant configs in %v/%v: %v", params.Svcs.Config.ConfigBucket, filepaths.RootDetectorConfig, err)
		return nil, err
	}

	// Return the names of the configs (dir names)
	configNamesFiltered := map[string]bool{}
	for _, path := range paths {
		bits := strings.Split(path, "/")
		if len(bits) > 2 {
			configNamesFiltered[bits[1]] = true
		}
	}

	// Form a list
	result := detectorConfigListing{ConfigNames: []string{}}
	for path := range configNamesFiltered {
		result.ConfigNames = append(result.ConfigNames, path)
	}

	sort.Strings(result.ConfigNames)
	return &result, err
}

func piquantConfigVersionsList(params handlers.ApiHandlerParams) (interface{}, error) {
	configName := params.PathParams[idIdentifier]

	// Get a list of PIQUANT config versions too
	versions := piquant.GetPiquantConfigVersions(params.Svcs, configName)

	return versions, nil
}

func piquantConfigGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// It's a get, we don't care about the body...

	// Using path params, work out path
	configName := params.PathParams[idIdentifier]
	version := params.PathParams[versionIdentifier]

	// Download configs
	detectorConfig, err := detector.ReadDetectorConfig(params.Svcs, configName)
	if err != nil {
		return nil, err
	}

	quantCfg, err := piquant.GetPIQUANTConfig(params.Svcs, configName, version)
	if err != nil {
		return nil, err
	}

	// Combine & return
	result := quantConfig{PixliseConfig: detectorConfig, QuantConfig: quantCfg}
	return &result, nil
}

func piquantDownloadList(params handlers.ApiHandlerParams) (interface{}, error) {
	// Return a list of all piquant builds we have stored
	resp, err := params.Svcs.S3.ListObjectsV2(
		&s3.ListObjectsV2Input{
			Bucket: aws.String(params.Svcs.Config.BuildsBucket),
			Prefix: aws.String(filepaths.PiquantDownloadPath + "/"),
		})

	if err != nil {
		params.Svcs.Log.Errorf("Failed to list piquant configs in %v/%v: %v", params.Svcs.Config.BuildsBucket, filepaths.PiquantDownloadPath, err)
		return nil, err
	}

	result := piquantDownloadListing{[]piquantDownloadable{}}
	for _, item := range resp.Contents {
		if item.Key != nil && item.LastModified != nil && item.Size != nil {
			fileName := filepath.Base(*item.Key)
			fileNameNoExt := strings.TrimSuffix(fileName, ".zip")
			fileNameBits := strings.Split(fileNameNoExt, "-")

			// Expecting fileNameNoExt to be: piquant-linux-1.2.3
			if len(fileNameBits) == 3 && (fileNameBits[1] == "windows" || fileNameBits[1] == "linux") {
				// Generate signed URL so it can be downloaded directly from S3
				url, err := params.Svcs.Signer.GetSignedURL(params.Svcs.S3, params.Svcs.Config.BuildsBucket, *item.Key, config.PiquantDownloadSignedURLExpirySec)
				if err != nil {
					return nil, err
				}

				result.DownloadItems = append(result.DownloadItems, piquantDownloadable{
					BuildVersion:  fileNameBits[2],
					BuildDate:     item.LastModified.Unix(),
					FileName:      fileName,
					FileSizeBytes: *item.Size,
					DownloadLink:  url,
					OS:            fileNameBits[1],
				})
			}
		}
	}

	return &result, nil
}

func piquantVersionGet(params handlers.ApiHandlerParams) (interface{}, error) {
	ver, err := piquant.GetPiquantVersion(params.Svcs)
	if err != nil {
		return nil, api.MakeStatusError(http.StatusNotFound, errors.New("PIQUANT version not found"))
	}

	return &ver, nil
}

type piquantVersionConfigPost struct {
	Version string `json:"version"`
}

func piquantVersionPost(params handlers.ApiHandlerParams) (interface{}, error) { // Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	verBody := piquantVersionConfigPost{}
	err = json.Unmarshal(body, &verBody)
	if err != nil {
		return nil, err
	}

	verSave := piquant.PiquantVersionConfig{
		Version:            verBody.Version,
		ChangedUnixTimeSec: params.Svcs.TimeStamper.GetTimeNowSec(),
		Creator:            params.UserInfo,
	}

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.ConfigBucket, filepaths.GetConfigFilePath(filepaths.PiquantVersionFileName), &verSave)
}
