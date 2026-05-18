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

package quantRunner

func EstimateNodeCount(spectraCount uint, elementCount uint, desiredRunTimeSec uint, coresPerNode uint, maxNodes uint) uint {
	// Nodes = Spectra*(Elements+3) / 3*(RuntimeDesired * Cores)
	// See unit test for why...

	// Add 0.5 to round up, can't have it fractional
	nodeCount := uint((float32(spectraCount*(elementCount+3)) / float32(3*desiredRunTimeSec*coresPerNode)) + 0.5)

	// Clamp it to reasonable values
	if nodeCount < 1 {
		nodeCount = 1
	}
	// Don't go way overboard either
	if maxNodes > 0 && nodeCount > maxNodes {
		nodeCount = maxNodes
	}

	return nodeCount
}

func FilesPerNode(spectraCount uint, nodeCount uint) uint {
	// NOTE: this may result in some extra if the spectra don't divide exactly per node. Even for a single
	// node it'll generate+1, but that's ok, this is a limit, when generating PMC files, this will be ok
	return uint((float32(spectraCount) / float32(nodeCount)) + 0.5) //+ 1 // TODO: the +1 might be redundant here??
}
