package sdfToRSI

import (
	"fmt"
	"strings"
)

func Example_makeWriteSCLK() {
	fmt.Println(makeWriteSCLK( /*208601602,*/ "2022-301T14:31:18"))

	// Output:
	// 2AEDFBB7
}

func Example_checkMCCDetector() {
	// The "2A" here is detector B
	det, ok, err := checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01732A13  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// A
	det, ok, err = checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01732513  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// 0
	det, ok, err = checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01730013  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// Invalid detector
	det, ok, err = checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01733A13  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// Different line
	det, ok, err = checkMCCDetector("00382 : 320A0896  14320032  0A0000C8  32640002  01732A13  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// Words missing
	det, ok, err = checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01732A13  FFFF2FFF   FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// Word wrong
	det, ok, err = checkMCCDetector("00384 : 320A0896  14320032  0A0000C8  32640002  01732A1  FFFF2FFF  00000000  FFFF0000 ")
	fmt.Printf("%v|%v|%v\n", det, ok, err)

	// Output:
	// B|true|<nil>
	// A|true|<nil>
	// 0|true|<nil>
	// |true|Invalid detector: 3A
	// |false|<nil>
	// |true|Failed to read detector config
	// |true|Read invalid word: 01732A1
}

func Example_readIntBetween() {
	v, f, pos, err := readNumBetween("Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3", "Residual:", " ", read_int)
	fmt.Printf("%v|%v|%v|%v\n", v, f, pos, err)

	v, f, pos, err = readNumBetween("Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3", "Current: ", " ", read_int)
	fmt.Printf("%v|%v|%v|%v\n", v, f, pos, err)

	v, f, pos, err = readNumBetween("The Reference: 0x00A3 -- Sclk: 15:42:17   ---> Flags: 0x300E", "Reference: 0x", " ", read_int)
	fmt.Printf("%v|%v|%v|%v\n", v, f, pos, err)

	v, f, pos, err = readNumBetween("The Reference: 0x00A3 -- Sclk: 15:42:17   ---> Flags: 0x300E", "Reference: 0x", " ", read_int_hex)
	fmt.Printf("%v|%v|%v|%v\n", v, f, pos, err)

	v, f, pos, err = readNumBetween("VPS -           HVMON:  27.84 Kv |   5 volt pos:   5.00 V |        SDF Retry:     0", "HVMON:", " ", read_float)
	fmt.Printf("%v|%v|%v|%v\n", v, f, pos, err)

	// Output:
	// 3|0|79|<nil>
	// 283|0|47|<nil>
	// 0|0|0|failed to read int value after 'Reference: 0x'
	// 163|0|21|<nil>
	// 0|27.84|27|<nil>
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

	f, r, e = readFloat("2.L3    0.15369324    0.24655464        ")
	fmt.Printf("%v|%v|%v\n", f, e, r)

	// Output:
	// -0.12470774|<nil>|0.15369324    0.24655464
	// 0|Error: strconv.ParseFloat: parsing "2.L3": invalid syntax|0.15369324    0.24655464
}

func Example_takeToken() {
	tok, l, ok := takeToken("gv - 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 ", " -")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)
	tok, l, ok = takeToken("gv - 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 ", " ")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	tok, l, ok = takeToken(" 0x00dd40 : 00000000 00000000 00000000 00000000", " ")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	tok, l, ok = takeToken(strings.TrimLeft(" 0x00dd40 : 00000000 00000000 00000000 00000000", " "), " ")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	tok, l, ok = takeToken("abc", "b")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	tok, l, ok = takeToken("abc", "a")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	tok, l, ok = takeToken("a", "a")
	fmt.Printf("%v|%v|%v\n", tok, l, ok)

	// Output:
	// gv|0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 |true
	// gv|- 0x00dd40 : 00000000 00000000 00000000 00000000 ::             0             0             0             0 |true
	// 0x00dd40|: 00000000 00000000 00000000 00000000|true
	// 0x00dd40|: 00000000 00000000 00000000 00000000|true
	// a|c|true
	// bc||true
	// ||false
}
