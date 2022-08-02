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
