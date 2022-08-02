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
	return int32((float32(spectraCount)/float32(nodeCount))+0.5) + 1 // TODO: the +1 might be redundant here??
}
