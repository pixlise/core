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

import (
	"fmt"

	"github.com/pixlise/core/core/pixlUser"
)

// Making sure our embedded structure copying works
func Example_makeJobStartingParametersWithPMCCount() {
	in := JobStartingParametersWithPMCs{
		[]int32{4, 59, 444, 2313, 329},
		&JobStartingParameters{
			Name:       "name",
			DataBucket: "databucket",
			//DatasetsBucket:    "datasetsbucket",
			//ConfigBucket:      "configbucket",
			DatasetPath:       "datasetpath",
			DatasetID:         "datasetid",
			PiquantJobsBucket: "jobbucket",
			DetectorConfig:    "config",
			Elements:          []string{"Ti", "Al", "Ca"},
			Parameters:        "params",
			RunTimeSec:        39,
			CoresPerNode:      3,
			StartUnixTime:     33332222,
			Creator: pixlUser.UserInfo{
				Name:        "creator",
				UserID:      "creator-id-123",
				Email:       "niko@rockstar.com",
				Permissions: map[string]bool{"read:something": true, "read:another": true},
			},
			RoiID:          "roiID",
			ElementSetID:   "elemSetID",
			PIQUANTVersion: "3.0.3",
		},
	}

	out := MakeJobStartingParametersWithPMCCount(in)
	fmt.Printf("pmc=%v, name=%v, elements=%v", out.PMCCount, out.Name, out.Elements)

	// Output:
	// pmc=5, name=name, elements=[Ti Al Ca]
}
