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

package dataset

import (
	"fmt"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
)

// SummaryFileData - Structure of dataset summary JSON files
type SummaryFileData struct {
	DatasetID           string      `json:"dataset_id"`
	Group               string      `json:"group"`
	DriveID             int32       `json:"drive_id"`
	SiteID              int32       `json:"site_id"`
	TargetID            string      `json:"target_id"`
	Site                string      `json:"site"`
	Target              string      `json:"target"`
	Title               string      `json:"title"`
	SOL                 string      `json:"sol"`
	RTT                 interface{} `json:"rtt,string"` // Unfortunately we stored it as int initially, so this has to accept files stored that way
	SCLK                int32       `json:"sclk"`
	ContextImage        string      `json:"context_image"`
	LocationCount       int         `json:"location_count"`
	DataFileSize        int         `json:"data_file_size"`
	ContextImages       int         `json:"context_images"`
	TIFFContextImages   int         `json:"tiff_context_images"`
	NormalSpectra       int         `json:"normal_spectra"`
	DwellSpectra        int         `json:"dwell_spectra"`
	BulkSpectra         int         `json:"bulk_spectra"`
	MaxSpectra          int         `json:"max_spectra"`
	PseudoIntensities   int         `json:"pseudo_intensities"`
	DetectorConfig      string      `json:"detector_config"`
	CreationUnixTimeSec int64       `json:"create_unixtime_sec"`
}

func (s SummaryFileData) GetRTT() string {
	result := ""
	switch s.RTT.(type) {
	case float64:
		f, ok := s.RTT.(float64)
		if ok {
			result = fmt.Sprintf("%d", int(f))
		}
	case int:
		i, ok := s.RTT.(int)
		if ok {
			result = fmt.Sprintf("%d", i)
		}
	default:
		result = fmt.Sprintf("%v", s.RTT)
	}

	padding := 9 - len(result)
	if padding > 0 {
		for i := 0; i < padding; i++ {
			result = "0" + result
		}
	}
	return result
}

// DatasetConfig is the container of the above
// This is the struct for dataset JSON files, as used by datasource updater lambda and dataset listing API endpoint
type DatasetConfig struct {
	Datasets []SummaryFileData `json:"datasets"`
}

// APIDatasetSummary - contains metadata fields for a given dataset
// This is returned from the API dataset listing endpoint. It's not private to that code
// because it's also used in the integration test code that tests it, and may be needed elsewhere in future
type APIDatasetSummary struct {
	*SummaryFileData

	DataSetLink      string `json:"dataset_link"`
	ContextImageLink string `json:"context_image_link"`
}

func ReadDataSetSummary(fs fileaccess.FileAccess, dataBucket string, datasetID string) (SummaryFileData, error) {
	result := SummaryFileData{}
	s3Path := filepaths.GetDatasetFilePath(datasetID, filepaths.DatasetSummaryFileName)

	err := fs.ReadJSON(dataBucket, s3Path, &result, false)
	if err == nil {
		result.RTT = result.GetRTT() // For backwards compatibility we can read it as an int, but here we convert to string
	}
	return result, err
}

// DatasetAuthInfo - Structure of dataset auth JSON files
// This is used to check whether an individual dataset is public or not
type DatasetAuthInfo struct {
	DatasetID               string `json:"dataset_id"`
	Public                  bool   `json:"public"`
	PublicReleaseUTCTimeSec int64  `json:"public_release_utc_time_sec"`
	Sol                     string `json:"sol"`
}

// DatasetsAuth - Structure of dataset auth JSON files
// This is used to check the public status of all datasets
type DatasetsAuth map[string]DatasetAuthInfo
