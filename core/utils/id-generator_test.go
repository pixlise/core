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

package utils

import (
	"fmt"
	"math/rand"
	"time"
)

// Checking that our IDs are pretty random...
func Example_genObjectID() {
	rand.Seed(time.Now().UnixNano())

	ids := map[string]bool{}

	var g IDGen

	for c := 0; c < 1000000; c++ {
		id := g.GenObjectID()
		_, exists := ids[id]
		if exists {
			fmt.Printf("match: %v\n", c)
			break
		}
		ids[id] = true
	}

	fmt.Println("done")

	// Output:
	// done
}
