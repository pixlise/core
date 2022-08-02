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
	"fmt"

	protos "github.com/pixlise/core/generated-protos"
)

// GetDetectorMetaValue - Has to do the same job as the function with the same name in PIXLISE DataSets.ts:
func GetDetectorMetaValue(metaLabel string, detector *protos.Experiment_Location_DetectorSpectrum, dataset *protos.Experiment) (protos.Experiment_MetaDataType, *protos.Experiment_Location_MetaDataItem, error) {
	// Look up the label
	c := int32(0)
	found := false
	for _, label := range dataset.MetaLabels {
		if label == metaLabel {
			found = true
			break
		}
		c++
	}

	if !found {
		return protos.Experiment_MT_FLOAT, nil, fmt.Errorf("Failed to find detector meta value: %v", metaLabel)
	}

	typeStored := dataset.MetaTypes[c]

	for _, metaValue := range detector.Meta {
		if metaValue.LabelIdx == c {
			return typeStored, metaValue, nil
		}
	}

	return protos.Experiment_MT_FLOAT, nil, fmt.Errorf("Failed to read detector meta value %v for type: %v", metaLabel, typeStored)
}

// Intermediate representation which can be passed as an array to spectrum CSV writer. This is useful to be able to provide bulk sums!
type SpectrumMetaValues struct {
	SCLK     int32
	RealTime float32
	LiveTime float32
	XPerChan float32
	Offset   float32
	Detector string
	ReadType string
}

// GetSpectrumMeta - Read meta columns saved in our bin file for a given spectrum
// These are named in pixlise-data-converter/importer/pixlfm/spectra.go
func GetSpectrumMeta(detector *protos.Experiment_Location_DetectorSpectrum, dataset *protos.Experiment) (SpectrumMetaValues, error) {
	result := SpectrumMetaValues{}

	metaField := []string{"DETECTOR_ID", "READTYPE", "SCLK", "REALTIME", "LIVETIME", "XPERCHAN", "OFFSET"}
	metaExpectedType := []protos.Experiment_MetaDataType{
		protos.Experiment_MT_STRING,
		protos.Experiment_MT_STRING,
		protos.Experiment_MT_INT,
		protos.Experiment_MT_FLOAT,
		protos.Experiment_MT_FLOAT,
		protos.Experiment_MT_FLOAT,
		protos.Experiment_MT_FLOAT,
	}
	metaVars := []*protos.Experiment_Location_MetaDataItem{}
	metaReadFailOK := []bool{false, false, true, true, true, true, true}

	for c, field := range metaField {
		metaType, metaVar, err := GetDetectorMetaValue(field, detector, dataset)
		// If we failed to read it and it's a crucial value, we stop here
		if err != nil && !metaReadFailOK[c] {
			return result, err
		}

		// If read was OK and the type is wrong, check that (if read was NOT ok and this is an issue, we wouldn't be here)
		if err == nil && metaType != metaExpectedType[c] {
			return result, fmt.Errorf("Expected %v to be %v when writing spectra", field, metaExpectedType[c])
		}

		metaVars = append(metaVars, metaVar)
	}

	// Now that we've read them, and verified they're of the right type, we can return a result
	if metaVars[0] != nil {
		result.Detector = metaVars[0].Svalue
	}
	if metaVars[1] != nil {
		result.ReadType = metaVars[1].Svalue
	}
	if metaVars[2] != nil {
		result.SCLK = metaVars[2].Ivalue
	}
	if metaVars[3] != nil {
		result.RealTime = metaVars[3].Fvalue
	}
	if metaVars[4] != nil {
		result.LiveTime = metaVars[4].Fvalue
	}
	if metaVars[5] != nil {
		result.XPerChan = metaVars[5].Fvalue
	}
	if metaVars[6] != nil {
		result.Offset = metaVars[6].Fvalue
	}

	return result, nil
}
