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

package pixlfm

import (
	"fmt"

	"github.com/pixlise/core/v3/core/gdsfilename"
	"github.com/pixlise/core/v3/core/logger"
)

func Example_GetByLowestSCLK() {
	files := []string{
		"PS__0866_0743796726_000RFS__04200003036943412155___J01_2582.MSA",
		"PS__0865_0743764558_000RFS__04200003036943410004___J01.CSV",
		"PS__0866_0743796726_000RFS__04200003036943412155___J01_2462.MSA",
		"PS__0866_0743796726_000RFS__04200003036943412155___J01.CSV",
		"PS__0866_0743796726_000RFS__04200003036943412155___J01_2154.MSA",
	}

	fileMetas := gdsfilename.GetLatestFileVersions(files, &logger.NullLogger{})

	fmt.Println(getByLowestSCLK(fileMetas))

	// Output:
	// PS__0865_0743764558_000RFS__04200003036943410004___J01.CSV
}
