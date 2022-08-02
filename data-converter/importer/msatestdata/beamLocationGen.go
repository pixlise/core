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

package msatestdata

import (
	"fmt"

	"github.com/pixlise/core/data-converter/converterModels"
	protos "github.com/pixlise/core/generated-protos"
)

// This is a specific feature for msa test data to generate a beam location when there isn't one
func makeBeamLocationFromSpectrums(lookup converterModels.DetectorSampleByPMC, params map[string]float32, minContextPMC int32) (converterModels.BeamLocationByPMC, error) {
	var beamLookup = converterModels.BeamLocationByPMC{}

	for pmc, items := range lookup {
		specMeta := items[0].Meta
		if specMeta["XPOSITION"].DataType != protos.Experiment_MT_FLOAT || specMeta["YPOSITION"].DataType != protos.Experiment_MT_FLOAT || specMeta["ZPOSITION"].DataType != protos.Experiment_MT_FLOAT {
			return beamLookup, fmt.Errorf("Error generating beam location, pmc: %v x/y/z position was not float", pmc)
		}
		x := specMeta["XPOSITION"].FValue
		y := specMeta["YPOSITION"].FValue
		z := specMeta["ZPOSITION"].FValue

		beamItem := converterModels.BeamLocation{
			X: x,
			Y: y,
			Z: z,
			IJ: map[int32]converterModels.BeamLocationProj{
				minContextPMC: converterModels.BeamLocationProj{
					I: x*params["xscale"] + params["xbias"],
					J: y*params["yscale"] + params["ybias"],
				},
			},
		}

		beamLookup[pmc] = beamItem
	}

	return beamLookup, nil
}
