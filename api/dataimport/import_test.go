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

package dataimport

import (
	"fmt"

	protos "github.com/pixlise/core/v4/generated-protos"
)

func Example_getUpdateType_NormalSpectra() {
	newSummary := protos.ScanItem{
		ContentCounts: map[string]int32{"NormalSpectra": 100},
	}
	oldSummary := protos.ScanItem{
		ContentCounts: map[string]int32{"NormalSpectra": 10},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// spectra|<nil>
}

func Example_getUpdateType_RTT() {
	newSummary := protos.ScanItem{
		Meta: map[string]string{"RTT": "1234"},
	}
	oldSummary := protos.ScanItem{
		Meta: map[string]string{"RTT": "123"},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// unknown|<nil>
}

func Example_getUpdateType_MoreContextImages() {
	newSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    3,
			},
		},
	}
	oldSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    0,
			},
		},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// image|<nil>
}

func Example_getUpdateType_LessContextImages() {
	newSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    3,
			},
		},
	}
	oldSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    5,
			},
		},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// image|<nil>
}

func Example_getUpdateType_SameContextImages() {
	newSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    3,
			},
		},
	}
	oldSummary := protos.ScanItem{
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			&protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    3,
			},
		},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// unknown|<nil>
}

func Example_getUpdateType_Drive() {
	newSummary := protos.ScanItem{
		Meta: map[string]string{"DriveID": "997"},
	}
	oldSummary := protos.ScanItem{
		Meta: map[string]string{"DriveID": "0"},
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// housekeeping|<nil>
}

func Example_getUpdateType_Title() {
	newSummary := protos.ScanItem{
		Title: "Analysed rock",
	}
	oldSummary := protos.ScanItem{
		Title: "Freshly downloaded rock",
	}

	upd, err := getUpdateType(&newSummary, &oldSummary)

	fmt.Printf("%v|%v\n", upd, err)

	// Output:
	// housekeeping|<nil>
}
