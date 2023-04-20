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

package permission

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pixlise/core/v3/core/api"
	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/pixlUser"
)

// We have a few public things, mainly getting the API version...
const PermPublic = "public"

// Permissions for routes - these should match the permissions defined
// in Auth0 JWT tokens that come in with requests

// Quantification creation
const PermCreateQuantification = "write:quantification"

// Quantification blessing
const PermBlessQuantification = "write:bless-quant"

// Quantification publishing
const PermPublishQuantification = "write:publish-quant"

// Piquant setup/administration
const PermReadPiquantConfig = "read:piquant-config"
const PermWritePiquantConfig = "write:piquant-config"
const PermDownloadPiquant = "download:piquant"
const PermReadDiffractionPeaks = "read:diffraction-peaks"
const PermEditDiffractionPeaks = "write:diffraction-peaks"

// Ability to export
const PermExportMap = "export:map"

// Permissions to view different kinds of datasets
const PermReadPIXLFullDataset = "read:pixl-full-dataset"
const PermReadPIXLTacticalDataset = "read:pixl-tactical-dataset"
const PermReadTestFullDataset = "read:test-full-dataset"
const PermReadTestTacticalDataset = "read:test-tactical-dataset"

// For reading ROI, element set, annotation, expressions
const PermReadDataAnalysis = "read:data-analysis"

// For being able to write/delete/edit the above
const PermWriteDataAnalysis = "write:data-analysis"

// For being able to edit custom fields/images on dataset
const PermWriteDataset = "write:dataset"

// General app permissions, eg saving/loading view state
const PermReadPIXLISESettings = "read:pixlise-settings"
const PermWritePIXLISESettings = "write:pixlise-settings"

// For saving metrics
const PermWriteMetrics = "write:metrics"

// Piquant jobs
const PermReadPiquantJobs = "read:piquant-jobs"

// User administration
const PermReadUserRoles = "read:user-roles"
const PermWriteUserRoles = "write:user-roles"

// Users own settings/name/data collection agreement
const PermReadUserSettings = "read:user-settings"
const PermWriteUserSettings = "write:user-settings"

// Sharing
const PermWriteSharedROI = "write:shared-roi"
const PermWriteSharedElementSet = "write:shared-element-set"
const PermWriteSharedQuantification = "write:shared-quantification"
const PermWriteSharedAnnotation = "write:shared-annotation"
const PermWriteSharedExpression = "write:shared-expression"

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

// Returns nil if user CAN access it, otherwise a api.StatusError with the right HTTP error code
func UserCanAccessDataset(userInfo pixlUser.UserInfo, summary datasetModel.SummaryFileData) error {
	userAllowedGroups := GetAccessibleGroups(userInfo.Permissions)
	if !userAllowedGroups[summary.Group] {
		// User is not allowed to see this
		return api.MakeStatusError(http.StatusForbidden, fmt.Errorf("dataset %v not permitted", summary.DatasetID))
	}
	return nil
}

func UserCanAccessDatasetWithSummaryDownload(fs fileaccess.FileAccess, userInfo pixlUser.UserInfo, dataBucket string, datasetID string) (datasetModel.SummaryFileData, error) {
	summary, err := datasetModel.ReadDataSetSummary(fs, dataBucket, datasetID)
	if err != nil {
		if fs.IsNotFoundError(err) {
			return summary, api.MakeNotFoundError(datasetID)
		} else {
			return summary, errors.New("failed to verify dataset group permission")
		}
	}

	return summary, UserCanAccessDataset(userInfo, summary)
}
