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

// Permission constants and helper functions for defining routes. These should match the permissions defined
// in Auth0 JWT tokens that come in with requests
package permission

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/api"
	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/utils"
)

// Public endpoints, mainly for getting the API version
const PermPublic = "public"

// Quantification creation
const PermCreateQuantification = "write:quantification"

// Quantification "blessing" - marking it as the correct one to use
const PermBlessQuantification = "write:bless-quant"

// Quantification publishing - to PDS
const PermPublishQuantification = "write:publish-quant"

// Reading piquant detector config and piquant config files
const PermReadPiquantConfig = "read:piquant-config"

// Writing piquant config (for spectroscopists who know what they're doing with piquant)
const PermWritePiquantConfig = "write:piquant-config"

// Downloading PIQUANT builds - not fully finished, likely only serving linux binaries if our build system still creates them
const PermDownloadPiquant = "download:piquant"

// Reading diffraction peaks DB that's created along with a dataset
const PermReadDiffractionPeaks = "read:diffraction-peaks"

// Editing diffraction peaks (manually creating new ones, or marking detected ones as deleted)
const PermEditDiffractionPeaks = "write:diffraction-peaks"

// Ability to export various data
const PermExportMap = "export:map"

// Reading ROI, element set, annotation, expressions, modules, tags, quantifications, RGB mixes
const PermReadDataAnalysis = "read:data-analysis"

// Write/delete/edit ROI, element set, annotation, expressions, modules, tags, quantifications, RGB mixes
const PermWriteDataAnalysis = "write:data-analysis"

// Allows editing custom fields/images on dataset, or creating new ones (using zipped MSA files, etc)
const PermWriteDataset = "write:dataset"

// Reading current view state, collections, workspaces
const PermReadPIXLISESettings = "read:pixlise-settings"

// Writing current view state, collections, workspaces
const PermWritePIXLISESettings = "write:pixlise-settings"

// Ability to call test endpoints (admin feature)
const PermTestEndpoints = "write:test-endpoints"

// For saving metrics - aka user tracking info, UI behaviours, for research purposes
const PermWriteMetrics = "write:metrics"

// Reading logs and log level of API
const PermReadLogs = "read:logs"

// Changing API log level (admin feature really!)
const PermWriteLogLevel = "write:log-level"

// Reads all piquant jobs - admin level
const PermReadPiquantJobs = "read:piquant-jobs"

// User role access - reading user listing, role listing and user/role individual gets
const PermReadUserRoles = "read:user-roles"

// Writing/deleting user roles, and editing users in bulk
const PermWriteUserRoles = "write:user-roles"

// Get users own config and data collection agreement
const PermReadUserSettings = "read:user-settings"

// Writing users own config and data collection agreement
const PermWriteUserSettings = "write:user-settings"

// Sharing ROI
const PermWriteSharedROI = "write:shared-roi"

// Sharing element sets
const PermWriteSharedElementSet = "write:shared-element-set"

// Sharing quantifications
const PermWriteSharedQuantification = "write:shared-quantification"

// Sharing annotations (of spectrum chart)
const PermWriteSharedAnnotation = "write:shared-annotation"

// Sharing expressions
const PermWriteSharedExpression = "write:shared-expression"

// Super Admin - not a real permission and mainly used to bypass tests
const PermSuperAdmin = "access:super-admin"

// Get all groups that are accessible by the list of permissions provided. This means
// basically returning what's after access: in each permission
func GetAccessibleGroups(permissions map[string]bool) map[string]bool {
	result := map[string]bool{}

	const accessPrefix = "access:"
	for perm := range permissions {
		// Make sure if the permission is just "access:", we don't store "" as a valid group
		if strings.HasPrefix(perm, accessPrefix) && len(perm) > len(accessPrefix) {
			group := perm[len(accessPrefix):]
			result[group] = true
		}
	}

	return result
}

func ReadPublicObjectsAuth(fs fileaccess.FileAccess, configBucket string, s3Path string) (PublicObjectsAuth, error) {
	publicObjectsAuth := PublicObjectsAuth{}

	err := fs.ReadJSON(configBucket, s3Path, &publicObjectsAuth, true)
	return publicObjectsAuth, err
}

func ReadDatasetsAuth(fs fileaccess.FileAccess, configBucket string, s3Path string) (DatasetsAuth, error) {
	datasetsAuth := DatasetsAuth{}
	err := fs.ReadJSON(configBucket, s3Path, &datasetsAuth, true)
	return datasetsAuth, err
}

func CheckAndUpdatePublicDataset(fs fileaccess.FileAccess, configBucket string, datasetID string, datasetsAuth DatasetsAuth) (bool, error) {
	isPublic := false
	datasetsAuthPath := filepaths.GetDatasetsAuthPath()

	// Check if it's in the public dict
	if datasetInfo, ok := datasetsAuth[datasetID]; ok {
		isPublic = datasetInfo.Public
		if !isPublic {
			// Check if it's past the date where the dataset should be released public
			if datasetInfo.PublicReleaseUTCTimeSec > 0 {
				isPublic = time.Now().Unix() > datasetInfo.PublicReleaseUTCTimeSec

				// If it's now public, update the public flag in the dict
				if isPublic {
					datasetInfo.Public = true
					err := fs.WriteJSON(configBucket, datasetsAuthPath, datasetsAuth)
					if err != nil {
						return isPublic, err
					}
				}
			}
		}
	}

	return isPublic, nil
}

// Check if the dataset CAN be public
func CheckIsPublicDataset(fs fileaccess.FileAccess, configBucket string, datasetID string) (bool, error) {
	isPublic := false

	datasetsAuthPath := filepaths.GetDatasetsAuthPath()
	datasetsAuth, err := ReadDatasetsAuth(fs, configBucket, datasetsAuthPath)
	if err != nil {
		return isPublic, err
	}

	return CheckAndUpdatePublicDataset(fs, configBucket, datasetID, datasetsAuth)
}

func GetPublicObjectsAuth(fs fileaccess.FileAccess, configBucket string, isPublicUser bool) (PublicObjectsAuth, error) {
	publicObjectsAuth := PublicObjectsAuth{}
	if !isPublicUser {
		return publicObjectsAuth, nil
	}

	publicObjectsPath := filepaths.GetPublicObjectsPath()
	publicObjectsAuth, err := ReadPublicObjectsAuth(fs, configBucket, publicObjectsPath)
	if err != nil {
		return publicObjectsAuth, err
	}

	return publicObjectsAuth, nil
}

// Check if the dataset is both public and has shared objects in it
func CheckIsPublicDatasetWithSharedObjects(fs fileaccess.FileAccess, configBucket string, datasetID string) (bool, error) {
	publicObjectsAuth, err := GetPublicObjectsAuth(fs, configBucket, true)
	if err != nil {
		return false, err
	}

	return utils.StringInSlice(datasetID, publicObjectsAuth.Datasets), nil
}

func CheckIsObjectPublic(fs fileaccess.FileAccess, configBucket string, objectType PublicObjectEnumType, objectID string) (bool, error) {
	publicObjectsPath := filepaths.GetPublicObjectsPath()
	publicObjectsAuth, err := ReadPublicObjectsAuth(fs, configBucket, publicObjectsPath)
	if err != nil {
		return false, err
	}

	switch objectType {
	case PublicObjectDataset:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Datasets, objectID)
	case PublicObjectROI:
		return CheckIsObjectInPublicSet(publicObjectsAuth.ROIs, objectID)
	case PublicObjectExpression:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Expressions, objectID)
	case PublicObjectModule:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Modules, objectID)
	case PublicObjectRGBMix:
		return CheckIsObjectInPublicSet(publicObjectsAuth.RGBMixes, objectID)
	case PublicObjectQuantification:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Quantifications, objectID)
	case PublicObjectCollection:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Collections, objectID)
	case PublicObjectWorkspace:
		return CheckIsObjectInPublicSet(publicObjectsAuth.Workspaces, objectID)
	default:
		return false, errors.New("unknown object type")
	}
}

func CheckIsObjectInPublicSet(publicObjectsList []string, objectID string) (bool, error) {
	strippedObjectID, _ := utils.StripSharedItemIDPrefix(objectID)

	for _, publicObjectID := range publicObjectsList {
		strippedPublicObjectID, _ := utils.StripSharedItemIDPrefix(publicObjectID)
		if strippedPublicObjectID == strippedObjectID {
			return true, nil
		}
	}

	return false, nil
}

// Returns nil if user CAN access it, otherwise a api.StatusError with the right HTTP error code
func UserCanAccessDataset(userInfo pixlUser.UserInfo, summary datasetModel.SummaryFileData, fs fileaccess.FileAccess, configBucket string) error {
	userAllowedGroups := GetAccessibleGroups(userInfo.Permissions)
	if !userAllowedGroups[summary.Group] {
		isPublic, err := CheckIsPublicDataset(fs, configBucket, summary.DatasetID)
		if err != nil {
			return err
		} else if isPublic {
			// Public dataset, anyone can access it
			return nil
		} else {
			// User is not allowed to see this
			return api.MakeStatusError(http.StatusForbidden, fmt.Errorf("dataset %v not permitted", summary.DatasetID))
		}
	}
	return nil
}

// Checking if the user can access a given dataset - use this if you don't already have summary info downloaded
func UserCanAccessDatasetWithSummaryDownload(fs fileaccess.FileAccess, userInfo pixlUser.UserInfo, dataBucket string, configBucket string, datasetID string) (datasetModel.SummaryFileData, error) {
	summary, err := datasetModel.ReadDataSetSummary(fs, dataBucket, datasetID)
	if err != nil {
		if fs.IsNotFoundError(err) {
			return summary, api.MakeNotFoundError(datasetID)
		} else {
			return summary, errors.New("failed to verify dataset group permission")
		}
	}

	return summary, UserCanAccessDataset(userInfo, summary, fs, configBucket)
}