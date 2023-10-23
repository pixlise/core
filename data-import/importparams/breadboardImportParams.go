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

package importparams

// We expect a JSON with these values in test datasets to provide us all required parameters
type BreadboardImportParams struct {
	DatasetID                 string  `json:"datasetid"`            // Dataset ID to output (affects output path and goes in summary file)
	Title                     string  `json:"title"`                // Title for this dataset
	TargetID                  string  `json:"targetid"`             // Target id to include in output
	Target                    string  `json:"target"`               // Target name include in output
	SiteID                    int32   `json:"siteid"`               // Site id to include in output
	Site                      string  `json:"site"`                 // Site name to include in output
	Group                     string  `json:"group"`                // Group the dataset will belong to
	BeamFile                  string  `json:"beamfile"`             // Beam location CSV path
	MsaBeamParams             string  `json:"beamparams"`           // Beam generation params if no beam location file
	HousekeepingFile          string  `json:"housekeeping"`         // Housekeeping CSV path
	ContextImgDir             string  `json:"contextimgdir"`        // Dir to find context images in
	MsaDir                    string  `json:"msadir"`               // Dir to load MSA files from
	PseudoIntensityCSVPath    string  `json:"pseudointensitycsv"`   // Pseudointensity CSV path
	IgnoreMSAFiles            string  `json:"ignoremsa"`            // MSA files to ignore
	SingleDetectorMSAs        bool    `json:"singledetectormsa"`    // Expecting single detector (1 column) MSA files
	GenPMCs                   bool    `json:"genpmcs"`              // Generate PMCs because it's an older test dataset without any
	ReadTypeOverride          string  `json:"readtype"`             // What to read MSAs as (normal vs dwell) because files arent named that way
	DetectorADuplicate        bool    `json:"detaduplicate"`        // Duplication of detector A to B, because test MSA only had 1 set of spectra
	GenBulkMax                bool    `json:"genbulkmax"`           // Generate bulk sum/max channel (because test dataset didnt come with one)
	DetectorConfig            string  `json:"detectorconfig"`       // Detector config that created this dataset, passed to PIQUANT when quantifying
	BulkQuantFile             string  `json:"bulkquantfile"`        // Bulk quantification file (for tactical datasets)
	XPerChanA                 float32 `json:"ev_xperchan_a"`        // eV calibration eV/channel (detector A)
	OffsetA                   float32 `json:"ev_offset_a"`          // eV calibration eV start offset (detector A)
	XPerChanB                 float32 `json:"ev_xperchan_b"`        // eV calibration eV/channel (detector B)
	OffsetB                   float32 `json:"ev_offset_b"`          // eV calibration eV start offset (detector B)
	ExcludeNormalDwellSpectra bool    `json:"exclude_normal_dwell"` // Hack for tactical datasets - load all MSAs to gen bulk sum, but dont save them in output
	SOL                       string  `json:"sol"`                  // Might as well be able to specify SOL. Needed for first spectrum dataset on SOL13
}
