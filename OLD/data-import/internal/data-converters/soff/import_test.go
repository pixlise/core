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

package soff

import "fmt"

func Example_readSOFF_valid() {
	xmlPath := "./test_data/valid.xml"
	soff, err := readSOFF(xmlPath)

	fmt.Printf("err:%v\n", err)
	fmt.Printf("%+v", soff)

	// Output:
	// err:<nil>
	// &{IdentificationArea:{Title:this is a PIXL test dataset generated manually on October 20, 2022 ProductClass:Product_Observational} FileArea:[{File:{FileName:pe__0300_0693591971_000r08__00900001042027530003___j04.csv} TableDelimited:[{LocalIdentifier:housekeeping_frame Offset:{Unit:byte Value:1132} Records:7108 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:pe__0300_0693591971_000rxl__00900001042027530003___j04.csv} TableDelimited:[{LocalIdentifier:Xray_beam_positions Offset:{Unit:byte Value:141} Records:3335 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:ps__0300_0693591971_000rbs__00900001042027530000___j04.msa} TableDelimited:[{LocalIdentifier:bulk_sum_histogram Offset:{Unit:byte Value:1131} Records:4096 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:ps__0300_0693591971_000rbs__00900001042027530000___j04.msa} TableDelimited:[{LocalIdentifier:max_value_histogram Offset:{Unit:byte Value:1110} Records:4096 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:ps__0300_0693593437_000rfs__00900001042027530004___j02.csv} TableDelimited:[{LocalIdentifier:histogram_housekeeping Offset:{Unit:byte Value:125} Records:3333 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0} {LocalIdentifier:histogram_position Offset:{Unit:byte Value:324036} Records:3333 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0} {LocalIdentifier:histogram_A Offset:{Unit:byte Value:493229} Records:3333 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0} {LocalIdentifier:histogram_B Offset:{Unit:byte Value:30555386} Records:3333 Description: RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:ps__0300_0693593438_000rpm__00900001042027530004___j02.csv} TableDelimited:[{LocalIdentifier:pseudointensity_map_metadata Offset:{Unit:byte Value:11} Records:3333 Description:PMC and position data associated with the pseudointensity table in this record RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0} {LocalIdentifier:pseudointensity_map Offset:{Unit:byte Value:141790} Records:3333 Description:Pseudointensity values for each measurement RecordDelimiter:Carriage-Return Line-Feed FieldDelimiter:Comma Fields:0 Groups:0}] EncodedImage:{Offset:{Unit: Value:0}}} {File:{FileName:pcw_0300_0693593326_000rcm_n00900001042027530003075j02.tif} TableDelimited:[] EncodedImage:{Offset:{Unit:byte Value:0}}} {File:{FileName:pcw_0300_0693593351_000rcm_n009000010420275300030luj02.tif} TableDelimited:[] EncodedImage:{Offset:{Unit:byte Value:0}}} {File:{FileName:pcw_0300_0693593423_000rcm_n00900001042027530004075j02.tif} TableDelimited:[] EncodedImage:{Offset:{Unit:byte Value:0}}} {File:{FileName:pcw_0301_0693643568_000rcm_n00900001042027533338075j02.tif} TableDelimited:[] EncodedImage:{Offset:{Unit:byte Value:0}}}] ObservationArea:{TimeCoordinates:{StartDateTime:2000-00-00T00:00:00.000Z StopDateTime:2050-00-00T00:00:00.000Z} InvestigationArea:{Name:M2020 Type:Mission} ObservingSystem:{ObservingSystemComponents:[{Name:M2020 Rover Type:Spacecraft} {Name:PIXL Type:Instrument}]} TargetIdentification:{Name:Mars Type:Planet}}}
}

func Example_readSOFF_corrupt() {
	xmlPath := "./test_data/corrupt.xml"
	soff, err := readSOFF(xmlPath)

	fmt.Printf("err:%v\n", err)
	fmt.Printf("%+v", soff)

	// Output:
	// err:XML syntax error on line 31: element <Identification_Area> closed by </Product_Observational>
	// <nil>
}

func Example_readSOFF_missing_table() {
	xmlPath := "./test_data/missing_table.xml"
	soff, err := readSOFF(xmlPath)

	fmt.Printf("err:%v\n", err)
	fmt.Printf("%+v", soff)

	// Output:
	// err:Missing table: pseudointensity_map
	// <nil>
}

func Example_readSOFF_duplicate_table() {
	xmlPath := "./test_data/duplicate_table.xml"
	soff, err := readSOFF(xmlPath)

	fmt.Printf("err:%v\n", err)
	fmt.Printf("%+v", soff)

	// Output:
	// err:Duplicate table: histogram_position
	// <nil>
}

func Example_readSOFF_missing_file() {
	xmlPath := "./test_data/missing_file.xml"
	soff, err := readSOFF(xmlPath)

	fmt.Printf("err:%v\n", err)
	fmt.Printf("%+v", soff)

	// Output:
	// err:No file for table: max_value_histogram
	// <nil>
}
