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

package dataConverter

import (
	"fmt"

	"github.com/pixlise/core/v2/core/dataset"
)

func Example_getUpdateType_NormalSpectra() {
	newSummary := dataset.SummaryFileData{
		NormalSpectra: 100,
	}
	oldSummary := dataset.SummaryFileData{
		NormalSpectra: 10,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// spectra|<nil>
}

func Example_getUpdateType_RTT() {
	newSummary := dataset.SummaryFileData{
		RTT: "1234",
	}
	oldSummary := dataset.SummaryFileData{
		RTT: 123,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// unknown|<nil>
}

func Example_getUpdateType_MoreContextImages() {
	newSummary := dataset.SummaryFileData{
		ContextImages: 3,
	}
	oldSummary := dataset.SummaryFileData{
		ContextImages: 0,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// image|<nil>
}

func Example_getUpdateType_LessContextImages() {
	newSummary := dataset.SummaryFileData{
		ContextImages: 3,
	}
	oldSummary := dataset.SummaryFileData{
		ContextImages: 5,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// image|<nil>
}

func Example_getUpdateType_SameContextImages() {
	newSummary := dataset.SummaryFileData{
		ContextImages: 3,
	}
	oldSummary := dataset.SummaryFileData{
		ContextImages: 3,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// unknown|<nil>
}

func Example_getUpdateType_Drive() {
	newSummary := dataset.SummaryFileData{
		DriveID: 997,
	}
	oldSummary := dataset.SummaryFileData{
		DriveID: 0,
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// housekeeping|<nil>
}

func Example_getUpdateType_Title() {
	newSummary := dataset.SummaryFileData{
		Title: "Analysed rock",
	}
	oldSummary := dataset.SummaryFileData{
		Title: "Freshly downloaded rock",
	}

	upd, err := getUpdateType(newSummary, oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// housekeeping|<nil>
}
