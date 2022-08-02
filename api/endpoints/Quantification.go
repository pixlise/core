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
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
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

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Quant URL path elements
const quantURLPathPrefix = "quantification"
const quantLogIdentifier = "logid"

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Router config
func registerQuantificationHandler(router *apiRouter.ApiObjectRouter) {
	// Used by piquant job admins
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadPiquantJobs), quantificationJobAdminList)

	// Normal users can access this - what quants are available and in-progress
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationList)

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
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationGet)

	// Deleting a quant
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), quantificationDelete)

	// "Blessing" a quant - a spectroscopist marking a given quant as "the one to use"
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, "bless", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermBlessQuantification), quantificationBless)

	// "Publishing" a quant - send to PDS
	router.AddJSONHandler(handlers.MakeEndpointPath(quantURLPathPrefix, "publish", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermPublishQuantification), quantificationPublish)

	// Sharing
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+quantURLPathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedQuantification), quantificationShare)

	// Streaming from S3
	router.AddStreamHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationFileStream)

	// Streaming log files from S3
	router.AddStreamHandler(handlers.MakeEndpointPath(quantURLPathPrefix+"/log/"+handlers.UrlStreamDownloadIndicator, datasetIdentifier, idIdentifier, quantLogIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), quantificationLogFileStream)
}
