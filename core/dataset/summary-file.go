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

package dataset

import (
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/fileaccess"
)

// SummaryFileData - Structure of dataset summary JSON files
type SummaryFileData struct {
	DatasetID           string `json:"dataset_id"`
	Group               string `json:"group"`
	DriveID             int32  `json:"drive_id"`
	SiteID              int32  `json:"site_id"`
	TargetID            string `json:"target_id"`
	Site                string `json:"site"`
	Target              string `json:"target"`
	Title               string `json:"title"`
	SOL                 string `json:"sol"`
	RTT                 int32  `json:"rtt"`
	SCLK                int32  `json:"sclk"`
	ContextImage        string `json:"context_image"`
	LocationCount       int    `json:"location_count"`
	DataFileSize        int    `json:"data_file_size"`
	ContextImages       int    `json:"context_images"`
	TIFFContextImages   int    `json:"tiff_context_images"`
	NormalSpectra       int    `json:"normal_spectra"`
	DwellSpectra        int    `json:"dwell_spectra"`
	BulkSpectra         int    `json:"bulk_spectra"`
	MaxSpectra          int    `json:"max_spectra"`
	PseudoIntensities   int    `json:"pseudo_intensities"`
	DetectorConfig      string `json:"detector_config"`
	CreationUnixTimeSec int64  `json:"create_unixtime_sec"`
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
	return result, fs.ReadJSON(dataBucket, s3Path, &result, false)
}
