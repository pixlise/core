package sdfToRSI

import (
	"fmt"
	"sort"
	"strings"
)

func Example_ConvertSDFtoRSI() {
	fmt.Printf("%v\n", ConvertSDFtoRSI("./test-data/sdf_raw.txt", "./output/RSI.csv"))

	// Output:
	// <nil>
}

func Example_makeWriteSCLK() {
	fmt.Println(makeWriteSCLK( /*208601602,*/ "2022-301T14:31:18"))

	// Output:
	// 2AEDFBB7
}

func Example_processCentroidLine1() {
	sliNum, x, y, intensity, err := processCentroidLine1(7, "", "3 -- pixel x,y,intensity: [0x7ab6fdcf] [0x37d1f047] [0x03f1] |     411.7626     187.3011")
	fmt.Printf("%v,%v,%v,%v,%v\n", sliNum, x, y, intensity, err)

	// Output:
	// 3,411.7626,187.3011,1009,<nil>
}

func Example_processCentroidLine2() {
	x, y, z, err := processCentroidLine2(7, "2022-301T17:28:38 :    2 CenSLI_struct  0 -- position x,y,z: [0xffffdbb9] [0xffffc999] [0x0000e8e3]  |    -0.009287    -0.013927     0.059619", 0)
	fmt.Printf("%v,%v,%v,%v\n", x, y, z, err)

	// Output:
	// -0.009287,-0.013927,0.059619,<nil>
}

func Example_processCentroidLine3() {
	id, res, err := processCentroidLine3(7, "2022-301T17:28:38 :    2 CenSLI_struct  0 -- ID: 0x4a, Residual: 0x03", 0)
	fmt.Printf("%v,%v,%v\n", id, res, err)

	// Output:
	// 74,3,<nil>
}

func Example_readIntBetween() {
	v, pos, err := readIntBetween("Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3", "Residual:", " ", false)
	fmt.Printf("%v|%v|%v\n", v, pos, err)

	v, pos, err = readIntBetween("Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3", "Current: ", " ", false)
	fmt.Printf("%v|%v|%v\n", v, pos, err)

	v, pos, err = readIntBetween("The Reference: 0x00A3 -- Sclk: 15:42:17   ---> Flags: 0x300E", "Reference: 0x", " ", false)
	fmt.Printf("%v|%v|%v\n", v, pos, err)

	v, pos, err = readIntBetween("The Reference: 0x00A3 -- Sclk: 15:42:17   ---> Flags: 0x300E", "Reference: 0x", " ", true)
	fmt.Printf("%v|%v|%v\n", v, pos, err)

	// Output:
	// 3|79|<nil>
	// 283|47|<nil>
	// 0|0|failed to read value after 'Reference: 0x'
	// 163|21|<nil>
}

func Example_findToken() {
	tok, ok := findToken("2022-302T02:28:56 : 1660 hk fcnt:11869 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEEA3E9 - Power Control: 0x08000408", "HK Time: ", " ")
	fmt.Printf("%v|%v\n", ok, tok)

	tok, ok = findToken("2022-302T02:28:56 : 1660 hk fcnt:11869 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEEA3E9", "HK Time: ", " ")
	fmt.Printf("%v|%v\n", ok, tok)

	tok, ok = findToken("HK Time: 0x2AEEA3E9 - Power Control: 0x08000408", "HK Time: ", " ")
	fmt.Printf("%v|%v\n", ok, tok)

	tok, ok = findToken("2022-302T02:28:56 : 1660 hk fcnt:11869 - FPGAVersion: 0x190425F4 - HK Ti me: 0x2AEEA3E9 - Power Control: 0x08000408", "HK Time: ", " ")
	fmt.Printf("%v|%v\n", ok, tok)

	tok, ok = findToken("2022-302T02:28:56 : 1660 hk fcnt:11869 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEEA3E9 - Power Control: 0x08000408", " - FPGAVersion: ", "F4")
	fmt.Printf("%v|%v\n", ok, tok)

	// Output:
	// true|0x2AEEA3E9
	// true|0x2AEEA3E9
	// true|0x2AEEA3E9
	// false|
	// true|0x190425
}

func Example_readFloat() {
	f, r, e := readFloat("-0.12470774    0.15369324    0.24655464        ")
	fmt.Printf("%v|%v|%v\n", f, e, strings.TrimRight(r, " "))

	f, r, e = readFloat(" -0.12470774    0.15369324    0.24655464        ")
	fmt.Printf("%v|%v|%v\n", f, e, r)

	// Output:
	// -0.12470774|<nil>|0.15369324    0.24655464
	// 0|Failed to read token| -0.12470774    0.15369324    0.24655464
}

func Example_takeToken() {
	tok, l, err := takeToken("gv - 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 ", " -")
	fmt.Printf("%v|%v|%v\n", tok, l, err)
	tok, l, err = takeToken("gv - 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 ", " ")
	fmt.Printf("%v|%v|%v\n", tok, l, err)

	tok, l, err = takeToken(" 0x00dd40 : 00000000 00000000 00000000 00000000", " ")
	fmt.Printf("%v|%v|%v\n", tok, l, err)

	tok, l, err = takeToken(strings.TrimLeft(" 0x00dd40 : 00000000 00000000 00000000 00000000", " "), " ")
	fmt.Printf("%v|%v|%v\n", tok, l, err)

	// Output:
	// gv|0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 |true
	// gv|- 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 |true
	// | 0x00dd40 : 00000000 00000000 00000000 00000000|false
	// 0x00dd40|: 00000000 00000000 00000000 00000000|true
}

func Example_scanForBasicStats() {
	rtt, sclk, err := scanForBasicStats("./test-data/BadPath.txt")
	fmt.Printf("%v|%v|%v\n", rtt, sclk, err)

	rtt, sclk, err = scanForBasicStats("./test-data/sdf_raw.txt")
	rtti := []int{}
	for _, r := range rtt {
		rtti = append(rtti, int(r))
	}
	rtti = sort.IntSlice(rtti)
	fmt.Printf("%v|%v|%v\n", rtt, sclk, err)

	// Output:
	// []|0|open ./test-data/BadPath.txt: The system cannot find the file specified.
	// [208536068 208536069 208601601 208601602]|720237093|<nil>
}
