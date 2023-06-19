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

package gdsfilename

import (
	"fmt"
	"sort"

	"github.com/pixlise/core/v3/core/logger"
)

func printFNValues(m FileNameMeta) {
	x, e := m.PMC()
	fmt.Printf("PMC=%v|%v\n", x, e)

	s, e := m.RTT()
	fmt.Printf("RTT=%v|%v\n", s, e)

	x, e = m.SCLK()
	fmt.Printf("SCLK=%v|%v\n", x, e)

	s, e = m.SOL()
	fmt.Printf("SOL=%v|%v\n\n", s, e)
}

func Example_parseFileName() {
	fmt.Println("Field test:")
	m, e := ParseFileName("INCSPRIMVSECONDARYT_TERPROGTSITDRIVSEQNUMRTTCAMSDCOPVE.EXT")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	fmt.Println("Invalid file name:")
	m, e = ParseFileName("hello.txt")
	fmt.Printf("%v\n\n", e)

	// pseudointensity file name example
	fmt.Println("Pseudo-intensity file:")
	m, e = ParseFileName("PS__D077T0637741109_000RPM_N001003600098356100640__J01.CSV")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// context image
	fmt.Println("Context image file:")
	m, e = ParseFileName("PCR_D077T0637741562_000EDR_N00100360009835610066000J01.PNG")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// bulk MSA
	fmt.Println("MSA (bulk) file:")
	m, e = ParseFileName("PS__D077T0637746318_000RBS_N001003600098356103760__J01.MSA")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// spatial inputs (housekeeping)
	fmt.Println("Housekeeping (RSI) file:")
	m, e = ParseFileName("PE__D077T0637741109_000RSI_N001003600098356100660__J01.CSV")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// spectra
	fmt.Println("All spectra CSV file:")
	m, e = ParseFileName("PS__D077T0637741109_000RFS_N001003600098356100640__J01.CSV")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// Something with SCLK and SOL
	fmt.Println("Testing SCLK and SOL fields:")
	m, e = ParseFileName("PS__1033_0012345678_000RFS_N001003600098356100640__J01.CSV")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// Watson image file
	// We see them as: CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01
	// But originally they are:
	fmt.Println("Watson file:")
	m, e = ParseFileName("SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01.PNG")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// Sherloc file
	fmt.Println("Sherloc file:")
	m, e = ParseFileName("SS__0614_0721475157_600RRS__0301172SRLC11360W208CGNJ01.CSV")
	fmt.Printf("%v|%v\n", e, m)
	printFNValues(m)

	// Output:
	// Field test:
	// <nil>|{IN C S PRIM V SECONDARYT TER PRO G T SIT DRIV SEQNUMRTT CAMS D CO P VE}
	// PMC=0|PMC only stored for PIXL files
	// RTT=SEQNUMRTT|<nil>
	// SCLK=0|Failed to get SCLK from: SECONDARYT
	// SOL=PRIM|<nil>
	//
	// Invalid file name:
	// Failed to parse meta from file name
	//
	// Pseudo-intensity file:
	// <nil>|{PS _ _ D077 T 0637741109 000 RPM _ N 001 0036 000983561 0064 0 __ J 01}
	// PMC=64|<nil>
	// RTT=000983561|<nil>
	// SCLK=637741109|<nil>
	// SOL=D077|<nil>
	//
	// Context image file:
	// <nil>|{PC R _ D077 T 0637741562 000 EDR _ N 001 0036 000983561 0066 0 00 J 01}
	// PMC=66|<nil>
	// RTT=000983561|<nil>
	// SCLK=637741562|<nil>
	// SOL=D077|<nil>
	//
	// MSA (bulk) file:
	// <nil>|{PS _ _ D077 T 0637746318 000 RBS _ N 001 0036 000983561 0376 0 __ J 01}
	// PMC=376|<nil>
	// RTT=000983561|<nil>
	// SCLK=637746318|<nil>
	// SOL=D077|<nil>
	//
	// Housekeeping (RSI) file:
	// <nil>|{PE _ _ D077 T 0637741109 000 RSI _ N 001 0036 000983561 0066 0 __ J 01}
	// PMC=66|<nil>
	// RTT=000983561|<nil>
	// SCLK=637741109|<nil>
	// SOL=D077|<nil>
	//
	// All spectra CSV file:
	// <nil>|{PS _ _ D077 T 0637741109 000 RFS _ N 001 0036 000983561 0064 0 __ J 01}
	// PMC=64|<nil>
	// RTT=000983561|<nil>
	// SCLK=637741109|<nil>
	// SOL=D077|<nil>
	//
	// Testing SCLK and SOL fields:
	// <nil>|{PS _ _ 1033 _ 0012345678 000 RFS _ N 001 0036 000983561 0064 0 __ J 01}
	// PMC=64|<nil>
	// RTT=000983561|<nil>
	// SCLK=12345678|<nil>
	// SOL=1033|<nil>
	//
	// Watson file:
	// <nil>|{SI F _ 0614 _ 0721455441 734 RAS _ N 030 1172 SRLC00643 _000 0 LM J 01}
	// PMC=0|PMC only stored for PIXL files
	// RTT=SRLC00643|<nil>
	// SCLK=721455441|<nil>
	// SOL=0614|<nil>
	//
	// Sherloc file:
	// <nil>|{SS _ _ 0614 _ 0721475157 600 RRS _ _ 030 1172 SRLC11360 W208 C GN J 01}
	// PMC=0|PMC only stored for PIXL files
	// RTT=SRLC11360|<nil>
	// SCLK=721475157|<nil>
	// SOL=0614|<nil>
}

func Example_stringFileName() {
	name := "PS__D077T0637741109_000RPM_N001003600098356100640__J01.CSV"
	m, e := ParseFileName(name)
	fmt.Printf("%v|%v\n", e, m.ToString())
	// Output:
	// <nil>|PS__D077T0637741109_000RPM_N001003600098356100640__J01
}

func Example_stringToIDSimpleCase() {
	i, b := stringToIDSimpleCase("123")
	fmt.Printf("%v|%v\n", i, b)
	i, b = stringToIDSimpleCase("12.3")
	fmt.Printf("%v|%v\n", i, b)
	i, b = stringToIDSimpleCase("0x32")
	fmt.Printf("%v|%v\n", i, b)
	i, b = stringToIDSimpleCase("i12")
	fmt.Printf("%v|%v\n", i, b)
	i, b = stringToIDSimpleCase("12i")
	fmt.Printf("%v|%v\n", i, b)

	// Output:
	// 123|true
	// 0|false
	// 0|false
	// 0|false
	// 0|false
}

func Example_isAllDigits() {
	fmt.Printf("%v\n", isAllDigits("1234567890"))
	fmt.Printf("%v\n", isAllDigits("9"))
	fmt.Printf("%v\n", isAllDigits("0"))
	fmt.Printf("%v\n", isAllDigits("01"))
	fmt.Printf("%v\n", isAllDigits("10"))
	fmt.Printf("%v\n", isAllDigits("12x4"))
	fmt.Printf("%v\n", isAllDigits("12.4"))

	// Output:
	// true
	// true
	// true
	// true
	// true
	// false
	// false
}

func Example_isAlpha() {
	fmt.Printf("%v\n", isAlpha('0'))
	fmt.Printf("%v\n", isAlpha('1'))
	fmt.Printf("%v\n", isAlpha('8'))
	fmt.Printf("%v\n", isAlpha('9'))
	fmt.Printf("%v\n", isAlpha('a'))
	fmt.Printf("%v\n", isAlpha('f'))
	fmt.Printf("%v\n", isAlpha('z'))
	fmt.Printf("%v\n", isAlpha('A'))
	fmt.Printf("%v\n", isAlpha('L'))
	fmt.Printf("%v\n", isAlpha('Z'))
	fmt.Printf("%v\n", isAlpha('.'))
	fmt.Printf("%v\n", isAlpha(' '))
	fmt.Printf("%v\n", isAlpha('^'))

	// Output:
	// false
	// false
	// false
	// false
	// true
	// true
	// true
	// true
	// true
	// true
	// false
	// false
	// false
}

func Example_letterValue() {
	fmt.Printf("%v\n", letterValue('A'))
	fmt.Printf("%v\n", letterValue('B'))
	fmt.Printf("%v\n", letterValue('Z'))
	fmt.Printf("%v\n", letterValue(' '))
	fmt.Printf("%v\n", letterValue('a'))
	fmt.Printf("%v\n", letterValue('0'))

	// Output:
	// 0
	// 1
	// 25
	// -33
	// 32
	// -17
}

func Example_stringToSiteID() {
	strs := []string{
		"123",
		"B01",
		"AA9",
		"AB8",
		"ZZ9",
		"AAZ",
		"ZZZ",
		"0AA",
		"0BZ",
		"7CZ",
		"7DV",
		"7DW", // Out of range, max is 32767
		"6",
		"HELLO",
	}
	for _, s := range strs {
		i, e := stringToSiteID(s)
		fmt.Printf("%v|%v\n", i, e)
	}

	// Output:
	// 123|<nil>
	// 1101|<nil>
	// 3609|<nil>
	// 3618|<nil>
	// 10359|<nil>
	// 10385|<nil>
	// 27935|<nil>
	// 27936|<nil>
	// 27987|<nil>
	// 32745|<nil>
	// 32767|<nil>
	// 0|Failed to convert: 7DW to site ID
	// 0|Failed to convert: 6 to site ID
	// 0|Failed to convert: HELLO to site ID
}

func Example_stringToDriveID() {
	strs := []string{
		"0000",
		"1234",
		"9999",
		"A000",
		"B001",
		"Z000",
		"AZ99",
		"BB99",
		"LJ00",
		"LJ35",
		"LJ36", // Out of range, max is 65535
		"300",
		"A00",
		"ZAZA",
	}
	for _, s := range strs {
		i, e := stringToDriveID(s)
		fmt.Printf("%v|%v\n", i, e)
	}

	// Output:
	// 0|<nil>
	// 1234|<nil>
	// 9999|<nil>
	// 10000|<nil>
	// 11001|<nil>
	// 35000|<nil>
	// 38599|<nil>
	// 38799|<nil>
	// 65500|<nil>
	// 65535|<nil>
	// 0|Failed to convert: LJ36 to drive ID
	// 0|Failed to convert: 300 to drive ID
	// 0|Failed to convert: A00 to drive ID
	// 0|Failed to convert: ZAZA to drive ID
}

func Example_stringToVersion() {
	strs := []string{"01", "55", "99", "A0", "AZ", "BA", "BZ", "Z0", "Z9", "ZZ", "Test", "3"}
	for _, s := range strs {
		i, e := stringToVersion(s)
		fmt.Printf("%v|%v\n", i, e)
	}

	// Output:
	// 1|<nil>
	// 55|<nil>
	// 99|<nil>
	// 100|<nil>
	// 135|<nil>
	// 146|<nil>
	// 171|<nil>
	// 1000|<nil>
	// 1009|<nil>
	// 1035|<nil>
	// 0|Failed to convert: Test to version
	// 0|Failed to convert: 3 to version
}

func Example_getLatestFileVersions() {
	files := []string{
		"PE__D140_0654321403_000RXL_N001000011000045300330__J01.LBL",
		"PE__D140_0654321406_000RXL_N001000011000045300330__J01.LBL",
		"PE__D140_0654321408_000RXL_N001000011000045300330__J03.LBL",
		"PE__D140_0654321406_000RXL_N001000011000045300330__J03.LBL",
		"PE__D140_0654321406_000RXL_N001000011000045300330__J01.CSV",
		"PE__D140_0654321402_000RXL_N001000011000045300330__J02.CSV",
		"PE__D140_0654321406_000RXL_N001000011000045300330__J02.CSV",
		"PE__D140_0654321406_000RXL_N001000011000045300331__J02.CSV",
		"PE__D140_0654321406_000RXL_N001000011000045300331__J04.CSV",
		"PE__D140_0654321404_000RXL_N001000011000045300331__J04.CSV",
	}

	latests := GetLatestFileVersions(files, &logger.NullLogger{})

	versionStrs := []string{}
	for key := range latests {
		versionStrs = append(versionStrs, key)
	}

	sort.Strings(versionStrs)

	for _, file := range versionStrs {
		meta := latests[file]
		fmt.Printf("%v sclk=%v version=%v\n", file, meta.secondaryTimestamp, meta.versionStr)
	}

	// Output:
	// PE__D140_0654321402_000RXL_N001000011000045300330__J02.CSV sclk=0654321402 version=02
	// PE__D140_0654321404_000RXL_N001000011000045300331__J04.CSV sclk=0654321404 version=04
	// PE__D140_0654321406_000RXL_N001000011000045300330__J03.LBL sclk=0654321406 version=03
}
