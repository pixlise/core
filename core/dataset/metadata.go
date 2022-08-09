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
