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

package jplbreadboard

import (
	"fmt"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// This is a specific feature for msa test data to generate a beam location when there isn't one
func makeBeamLocationFromSpectrums(lookup dataConvertModels.DetectorSampleByPMC, params map[string]float32, minContextPMC int32) (dataConvertModels.BeamLocationByPMC, error) {
	var beamLookup = dataConvertModels.BeamLocationByPMC{}

	for pmc, items := range lookup {
		specMeta := items[0].Meta
		if specMeta["XPOSITION"].DataType != protos.Experiment_MT_FLOAT || specMeta["YPOSITION"].DataType != protos.Experiment_MT_FLOAT || specMeta["ZPOSITION"].DataType != protos.Experiment_MT_FLOAT {
			return beamLookup, fmt.Errorf("Error generating beam location, pmc: %v x/y/z position was not float", pmc)
		}
		x := specMeta["XPOSITION"].FValue
		y := specMeta["YPOSITION"].FValue
		z := specMeta["ZPOSITION"].FValue

		beamItem := dataConvertModels.BeamLocation{
			X: x,
			Y: y,
			Z: z,
			IJ: map[int32]dataConvertModels.BeamLocationProj{
				minContextPMC: dataConvertModels.BeamLocationProj{
					I: x*params["xscale"] + params["xbias"],
					J: y*params["yscale"] + params["ybias"],
				},
			},
		}

		beamLookup[pmc] = beamItem
	}

	return beamLookup, nil
}
