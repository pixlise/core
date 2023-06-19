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

func estimateNodeCount(spectraCount int32, elementCount int32, desiredRunTimeSec int32, coresPerNode int32, maxNodes int32) int32 {
	// Nodes = Spectra*(Elements+3) / 3*(RuntimeDesired * Cores)
	// See unit test for why...

	// Add 0.5 to round up, can't have it fractional
	nodeCount := int32((float32(spectraCount*(elementCount+3)) / float32(3*desiredRunTimeSec*coresPerNode)) + 0.5)

	// Clamp it to reasonable values
	if nodeCount < 1 {
		nodeCount = 1
	}
	// Don't go way overboard either
	if nodeCount > maxNodes {
		nodeCount = maxNodes
	}
	return nodeCount
}

func filesPerNode(spectraCount int32, nodeCount int32) int32 {
	// NOTE: this may result in some extra if the spectra don't divide exactly per node. Even for a single
	// node it'll generate+1, but that's ok, this is a limit, when generating PMC files, this will be ok
	return int32((float32(spectraCount)/float32(nodeCount))+0.5) + 1 // TODO: the +1 might be redundant here??
}
