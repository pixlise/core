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

package quantification

import (
	"fmt"
)

func Example_cleanLogName() {
	// Don't fix it...
	fmt.Println(cleanLogName("node00001_data.log"))
	// Do fix it...
	fmt.Println(cleanLogName("node00001.pmcs_stdout.log"))
	// Do fix it...
	fmt.Println(cleanLogName("NODE00001.PMCS_stdout.log"))

	// Output:
	// node00001_data.log
	// node00001_stdout.log
	// NODE00001_stdout.log
}
