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

package detection

import (
	"math"
	"testing"

	datasetModel "github.com/pixlise/core/core/dataset"
)

func TestProtobuf(t *testing.T) {
	protoDatasetParsed, err := datasetModel.ReadDatasetFile("./test-files/dataset.bin")

	if err != nil {
		t.Errorf("Failed to open file : \"%v\"\n", err)
	}

	datasetPeaks, err := ScanDataset(protoDatasetParsed)
	if err != nil {
		t.Errorf("Error encountered during scanning %v", err)
	}

	if len(datasetPeaks) == 0 {
		t.Error("Scan did not find any valid locations in dataset")
	}
	for loc, peaks := range datasetPeaks {
		if len(peaks) == 0 {
			t.Errorf("Did not find any peaks for location: %v", loc)
		}
		for _, peak := range peaks {
			if math.IsNaN(peak.EffectSize) {
				t.Errorf("Spectra at location %v has peak at channel %v with NaN effect", loc, peak.PeakChannel)
			} else if math.IsInf(peak.EffectSize, 0) {
				t.Errorf("Spectra at location %v has peak at channel %v with infinite effect", loc, peak.PeakChannel)
			} else if peak.EffectSize < 0.0 {
				t.Errorf("Spectra at location %v has peak at channel %v with negative effect", loc, peak.PeakChannel)
			}
		}
	}

	diffractionPB := BuildDiffractionProtobuf(protoDatasetParsed, datasetPeaks)
	err = SaveDiffractionProtobuf(diffractionPB, "./test-files/dataset-diffraction.bin")
	if err != nil {
		t.Errorf("Error encountered during saving of protobuf file: %v", err)
	}

	_, err = ParseDiffractionProtoBuf("./test-files/dataset-diffraction.bin")

	if err != nil {
		t.Errorf("Failed to load diffraction peak protobuf file, error: %v", err)
	}
}
