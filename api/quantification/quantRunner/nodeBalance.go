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

func EstimateNodeCount(spectraCount uint, elementCount uint, desiredRunTimeSec uint, maxNodes uint) uint {
	// Real world testing:
	// 23 elements, 1377 spectra ran as 50 jobs, 25 nodes in 600sec => total processing is 30,000sec => 1 spectrum takes 21.8
	// 4 elements, 1377 spectra ran as 32 jobs, 16 nodes in 250sec => total processing is 8,000sec => 1 spectrum takes 5.8
	// 1 elements, 1377 spectra ran as 20 jobs, 10 nodes in 234sec => total processing is 4,680sec => 1 spectrum takes 3.4

	// So the element contribution is:
	// 1 element    3.4sec
	// 4 elements   5.8sec
	// 23 elements 21.8sec
	// A fitting curve is T=0.015E^2 + 0.6E + 2.9
	// where T is sec per spectra, E is number of elements
	// So calculating how much time per spectrum:
	secPerSpectrum := 0.015*float64(elementCount)*float64(elementCount) + 0.6*float64(elementCount) + 2.9

	// Within the desired runtime, how many ways do we need to farm this out?
	estRuntimeSec := float64(spectraCount) * secPerSpectrum

	nodeCount := uint(estRuntimeSec / float64(desiredRunTimeSec))

	// Make sure it's within limits
	if nodeCount <= 0 {
		nodeCount = 1
	} else if nodeCount > maxNodes {
		nodeCount = maxNodes
	}

	return uint(nodeCount)
}

func FilesPerNode(spectraCount uint, nodeCount uint) uint {
	// NOTE: this may result in some extra if the spectra don't divide exactly per node. Even for a single
	// node it'll generate+1, but that's ok, this is a limit, when generating PMC files, this will be ok
	return uint((float32(spectraCount) / float32(nodeCount)) + 0.5) //+ 1 // TODO: the +1 might be redundant here??
}
