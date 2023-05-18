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
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// How quantification job data is stored/accessed:
//
// s3://PiquantJobsBucket/RootJobData/<dataset-id>/<job-id>/
//     PMCS files, parameters, go here. PIQUANT runner generates ./piquant-logs/ and ./output/ dir
//
// s3://PiquantJobsBucket/RootJobStatus/<dataset-id>/
//     API writes status files here, named using <job-id>JobStatusSuffix
//
// s3://PiquantJobsBucket/RootJobSummaries/
//     JobUpdater lambda reads data from RootJobStatus and RootJobData, writes summary list for
//     all jobs per dataset to a single file: <dataset-id>JobSummarySuffix
//
// s3://UsersBucket/RootUserContent/<user-id>/<dataset-id>/Quantification/
//     When quant completes, API copies the output files here:
//     - <job-id>-summary.json <-- JSON containing job summary
//     - <job-id>.bin <-- PROTOBUF encoded binary file
//     - <job-id>.csv <-- Raw map CSV that came from PIQUANT
//     - <job-id>-logs/ <-- Contains PIQUANT log files for job
//
// Queries we need to get job data:
//   GET quantification/ <-- admin list of quants, should show ALL quants
//   GET quantification/<dataset-id>/ <-- user list of quants for a given dataset
//   GET quantification/<dataset-id>/<job-id>/ <-- querying a specific quant
// permission.UserCanAccessDatasetWithSummaryDownload

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Quant URL path elements
const quantURLPathPrefix = "quantification"
const quantLogIdentifier = "logid"
const quantCmdOutputIdentifier = "cmdoutput"

// //////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Router config
func registerQuantificationHandler(router *apiRouter.ApiObjectRouter) {
	// Used by piquant job admins
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadPiquantJobs), quantificationJobAdminList)

	// Normal users can access this - what quants are available and in-progress
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), quantificationList)

	// Multi-quant comparison
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/comparison-for-roi", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermReadDataAnalysis), multiQuantificationComparisonPost)

	// Quant creation
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermCreateQuantification), quantificationPost)

	// Quant upload
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/upload", datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermCreateQuantification), quantificationUpload)

	// Multi-quant generation
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/combine-list", datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermCreateQuantification), quantificationCombineListLoad)
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/combine-list", datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermCreateQuantification), quantificationCombineListSave)

	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/combine", datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermCreateQuantification), quantificationCombine)

	// Accessing individual quant
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), quantificationGet)

	// Deleting a quant
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), quantificationDelete)

	// "Blessing" a quant - a spectroscopist marking a given quant as "the one to use"
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, "bless", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermBlessQuantification), quantificationBless)

	// "Publishing" a quant - send to PDS
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, "publish", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermPublishQuantification), quantificationPublish)

	// Sharing
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedQuantification), quantificationShare)

	// Streaming quant files from S3 (map command)
	router.AddStreamHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermPublic), quantificationFileStream)

	// Streaming log files from S3
	router.AddStreamHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/log/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier, quantLogIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationLogFileStream)

	// Streaming last output and log from S3 (for anything other than the map command)
	// idIdentifier is the piquant command
	// quantCmdOutputIdentifier is either log or data
	router.AddStreamHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/last/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier, quantCmdOutputIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationLastRunFileStream)
}
