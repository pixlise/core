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

package output

func rLEncode(data []int) []int {
	var encoded []int

	count := 0

	last := data[0]

	for _, val := range data {
		if last == val {
			count = count + 1
		} else {
			encoded = append(encoded, last)
			encoded = append(encoded, count)
			last = val
			count = 1
		}
	}

	encoded = append(encoded, last)
	encoded = append(encoded, count)

	return encoded
}

func zeroRunEncode(data []int64) []int32 {
	var encoded []int32
	count := 0
	init := false
	for _, val := range data {
		if val != 0 {
			if init {
				encoded = append(encoded, int32(0))
				encoded = append(encoded, int32(count))
				init = false
			}
			encoded = append(encoded, int32(val))

		} else {
			if !init {
				count = 0
				init = true
			}
			count = count + 1
		}
	}

	if init {
		encoded = append(encoded, 0)
		encoded = append(encoded, int32(count))
	}
	return encoded
}
