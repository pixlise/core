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

package permission

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/pixlise/core/core/api"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/pixlUser"
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
