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

package quantModel

import "github.com/pixlise/core/core/pixlUser"

// Structures, constants and functionality of a quantification job

// JobParamsFileName - File name of job params file
const JobParamsFileName = "params.json"

// JobStartingParameters - parameters to start the job, saved in S3 as JobParamsFileName for the job
type JobStartingParameters struct {
	Name              string            `json:"name"`
	DataBucket        string            `json:"dataBucket"`
	DatasetPath       string            `json:"datasetPath"`
	DatasetID         string            `json:"datasetID"`
	PiquantJobsBucket string            `json:"jobBucket"`
	DetectorConfig    string            `json:"detectorConfig"`
	Elements          []string          `json:"elements"`
	Parameters        string            `json:"parameters"`
	RunTimeSec        int32             `json:"runTimeSec"`
	CoresPerNode      int32             `json:"coresPerNode"`
	StartUnixTime     int64             `json:"startUnixTime"`
	Creator           pixlUser.UserInfo `json:"creator"`
	RoiID             string            `json:"roiID"`
	ElementSetID      string            `json:"elementSetID"`
	PIQUANTVersion    string            `json:"piquantVersion"`
	QuantMode         string            `json:"quantMode"`
	Comments          string            `json:"comments"`
	RoiIDs            []string          `json:"roiIDs"`
	IncludeDwells     bool              `json:"includeDwells,omitempty"`
	//DatasetsBucket    string            `json:"dataBucket"`
	//ConfigBucket      string            `json:"configBucket"`
}

// JobStatusValue - the type for our job status field
type JobStatusValue string

// JobStartingParametersWithPMCCount - summary version of JobStartingParametersWithPMCs, only storing PMC count
type JobStartingParametersWithPMCCount struct {
	PMCCount int32 `json:"pmcsCount"`
	*JobStartingParameters
}

// JobStatus - job status that gets saved to S3 as <job-id>JobStatusSuffix as it progresses
type JobStatus struct {
	JobID          string         `json:"jobId"`
	Status         JobStatusValue `json:"status"`
	Message        string         `json:"message"`
	EndUnixTime    int64          `json:"endUnixTime"`
	OutputFilePath string         `json:"outputFilePath"`
	PiquantLogList []string       `json:"piquantLogList"`
}

// JobCreateParams - Parameters to CreateJob()
type JobCreateParams struct {
	Name           string            `json:"name"`
	DatasetPath    string            `json:"datasetPath"`
	PMCs           []int32           `json:"pmcs"`
	Elements       []string          `json:"elements"`
	DetectorConfig string            `json:"detectorconfig"`
	Parameters     string            `json:"parameters"`
	RunTimeSec     int32             `json:"runtimesec"`
	RoiID          string            `json:"roiID"` // There is now a list of ROI IDs that can be provided too. More relevant with the QuantMode *Bulk options
	ElementSetID   string            `json:"elementSetID"`
	DatasetID      string            `json:"dataset_id"`
	Creator        pixlUser.UserInfo `json:"creator"`
	QuantMode      string            `json:"quantMode"`
	RoiIDs         []string          `json:"roiIDs"` // If QuantMode = *Bulk, this is used, pmcs is ignored.
	IncludeDwells  bool              `json:"includeDwells,omitempty"`
}

// Valid job status strings
const (
	JobStarting         JobStatusValue = "starting"
	JobPreparingNodes                  = "preparing_nodes"
	JobNodesRunning                    = "nodes_running"
	JobGatheringResults                = "gathering_results"
	JobComplete                        = "complete"
	JobError                           = "error"
)

// Full list of parameters to start a quantification - generated by quant creation endpoint, and saved in bucket
// for future users

// JobStartingParametersWithPMCs - When saving file with name: JobParamsFileName, we save all PMCs, but
// in further references to the job parameters, eg in job list summary (filepaths.JobSummarySuffix), we only
// store the PMC count
type JobStartingParametersWithPMCs struct {
	PMCs []int32 `json:"pmcs"`
	*JobStartingParameters
}

// MakeJobStartingParametersWithPMCCount - Converting full to summary version of struct
func MakeJobStartingParametersWithPMCCount(params JobStartingParametersWithPMCs) JobStartingParametersWithPMCCount {
	return JobStartingParametersWithPMCCount{
		PMCCount:              int32(len(params.PMCs)),
		JobStartingParameters: params.JobStartingParameters,
	}
}
