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

package datasetArchive

import "fmt"

func Example_decodeManualUploadPath() {
	f, p, e := decodeManualUploadPath("/dataset-addons/dataset123/custom-meta.json")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Contains subdir
	f, p, e = decodeManualUploadPath("/dataset-addons/dataset123/MATCHED/something.png")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Contains multiple subdir
	f, p, e = decodeManualUploadPath("/dataset-addons/dataset123/MATCHED/more/file.png")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Without leading /
	f, p, e = decodeManualUploadPath("dataset-addons/dataset123/MATCHED/more/image.png")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Path too short
	f, p, e = decodeManualUploadPath("/dataset-addons/the-dir/invalid.txt")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Path way too short
	f, p, e = decodeManualUploadPath("/dataset-addons/invalid.txt")
	fmt.Printf("%v, %v, %v\n", f, p, e)

	// Output:
	// custom-meta.json, [], <nil>
	// something.png, [MATCHED], <nil>
	// file.png, [MATCHED more], <nil>
	// image.png, [MATCHED more], <nil>
	// , [], Manual upload path invalid: dataset-addons/the-dir/invalid.txt
	// , [], Manual upload path invalid: dataset-addons/invalid.txt
}

func Example_decodeArchiveFileName() {
	// Just a simple one
	id, ts, e := DecodeArchiveFileName("161677829-12-06-2022-06-41-00.zip")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	// Should accept paths too but snip the path off!
	id, ts, e = DecodeArchiveFileName("/Archive/161677829-12-06-2022-06-41-00.zip")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	id, ts, e = DecodeArchiveFileName("data/161677829-12-06-2022-06-41-00.zip")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	// FAIL: just a timestamp
	id, ts, e = DecodeArchiveFileName("12-06-2022-06-41-00.zip")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	// FAIL: something else
	id, ts, e = DecodeArchiveFileName("readme.txt")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	// FAIL: something else with path
	id, ts, e = DecodeArchiveFileName("/Archive/readme.txt")
	fmt.Printf("%v, %v, %v\n", id, ts, e)

	// Output:
	// 161677829, 1655016060, <nil>
	// 161677829, 1655016060, <nil>
	// 161677829, 1655016060, <nil>
	// , 0, DecodeArchiveFileName "12-06-2022-06-41-00.zip" error: parsing time "06-2022-06-41-00": month out of range
	// , 0, DecodeArchiveFileName unexpected file name: readme.txt
	// , 0, DecodeArchiveFileName unexpected file name: /Archive/readme.txt
}

func Example_getOrderedArchiveFiles() {
	ordered, err := getOrderedArchiveFiles([]string{"161677829-12-06-2022-06-41-00.zip", "161677829-12-06-2022-06-42-00.zip", "161677829-12-06-2022-06-39-00.zip", "161677829-12-05-2022-06-40-00.zip"})
	fmt.Printf("%v, %v\n", ordered, err)

	ordered, err = getOrderedArchiveFiles([]string{"Archive/161677829-12-06-2022-06-41-00.zip", "Archive/161677829-12-06-2022-06-42-00.zip", "Archive/161677829-12-06-2022-06-39-00.zip", "161677829-12-05-2022-06-40-00.zip"})
	fmt.Printf("%v, %v\n", ordered, err)

	ordered, err = getOrderedArchiveFiles([]string{"161677829-12-06-2022-06-41-00.zip", "161677829-12-06-2022-06-42-00.zip", "161677829-12-06-2022-06-39-00.zip", "161677829-12-05-2022-24-40-00.zip"})
	fmt.Printf("%v, %v\n", ordered, err)

	ordered, err = getOrderedArchiveFiles([]string{"161677829-12-06-2022-06-41-00.zip", "161677829-12-06-2022-06-42-00.zip", "12-06-2022-06-39-00.zip", "161677829-12-05-2022-06-40-00.zip"})
	fmt.Printf("%v, %v\n", ordered, err)

	ordered, err = getOrderedArchiveFiles([]string{"161677829-12-06-2022-06-41-00.zip", "161677829-12-06-2022-06-42-00.zip", "readme.txt", "161677829-12-05-2022-06-40-00.zip"})
	fmt.Printf("%v, %v\n", ordered, err)

	ordered, err = getOrderedArchiveFiles([]string{})
	fmt.Printf("%v, %v\n", ordered, err)

	// Output:
	// [161677829-12-05-2022-06-40-00.zip 161677829-12-06-2022-06-39-00.zip 161677829-12-06-2022-06-41-00.zip 161677829-12-06-2022-06-42-00.zip], <nil>
	// [161677829-12-05-2022-06-40-00.zip Archive/161677829-12-06-2022-06-39-00.zip Archive/161677829-12-06-2022-06-41-00.zip Archive/161677829-12-06-2022-06-42-00.zip], <nil>
	// [], DecodeArchiveFileName "161677829-12-05-2022-24-40-00.zip" error: parsing time "12-05-2022-24-40-00": hour out of range
	// [], DecodeArchiveFileName "12-06-2022-06-39-00.zip" error: parsing time "06-2022-06-39-00": month out of range
	// [], DecodeArchiveFileName unexpected file name: readme.txt
	// [], <nil>
}
