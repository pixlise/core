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
			Command:        "map",
		},
	}

	out := MakeJobStartingParametersWithPMCCount(in)
	fmt.Printf("pmc=%v, name=%v, elements=%v", out.PMCCount, out.Name, out.Elements)

	// Output:
	// pmc=5, name=name, elements=[Ti Al Ca]
}
