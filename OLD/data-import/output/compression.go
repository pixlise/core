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
