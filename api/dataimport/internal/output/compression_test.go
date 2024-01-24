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

import "fmt"

func Example_rleEncode() {

	data := []int{0, 0, 4, 2, 2, 2, 3, 0}

	encoded := rLEncode(data)

	fmt.Printf("%+v\n", encoded)

	// Output:
	// [0 2 4 1 2 3 3 1 0 1]
}

func Example_zeroRunEncode() {
	data := []int64{0, 0, 4, 2, 0, 0, 0, 0, 3, 0}

	encoded := zeroRunEncode(data)

	fmt.Printf("%+v\n", encoded)

	// Output:
	// [0 2 4 2 0 4 3 0 1]
}
