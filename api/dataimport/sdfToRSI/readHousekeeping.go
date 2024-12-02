package sdfToRSI

import (
	"fmt"
	"strings"
)

/*
hk_data.csv contains:
2022-302T00:22:08,10329.0,419702260,720274993,1.0,0.0,0.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,21.06,14.17,14.13,-43.63,41.97,-29.93,-29.95,7.67,-9.42,-0.77,4.93,-4.91,-146.54,-146.52,3.17,-60.29,-59.4,-59.4,-39.7,-42.41,1.78,8.94,-8.84,3.11,7.63,-42.47,-10.31,-10.79,-12.14,1.11,-10.31,3.91,0.66,27.84,19.94,13.15,-13.15,5.0,-3.31,42.0,0.0,0.0,0.0,1651.0,2139.0,1856.0,1727.0,2119.0,2192.0,1604.0,0.0,0.0,0.0,1.0,0.0,0.0,1.0,1.0,0.0,1.0
2022-302T00:22:40,10333.0,419702260,720275025,1.0,0.0,0.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,21.0,14.1,14.13,-43.66,41.94,-30.0,-29.95,7.7,-9.65,-0.48,4.93,-4.93,-146.52,-146.49,3.17,-60.29,-59.21,-59.65,-39.74,-42.5,1.78,8.94,-8.85,3.12,7.65,-42.41,-10.64,-10.82,-12.08,1.15,-10.24,3.93,0.68,27.8,20.05,13.15,-13.2,5.0,-3.17,42.0,0.0,0.0,0.0,1651.0,2139.0,1856.0,1727.0,2119.0,2192.0,1604.0,0.0,0.0,0.0,1.0,0.0,0.0,1.0,1.0,0.0,1.0
2022-302T00:22:52,10335.0,419702260,720275037,1.0,0.0,0.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,1.0,20.97,14.13,14.1,-43.63,41.97,-30.01,-30.0,7.9,-9.71,-0.22,4.92,-4.9,-146.52,-146.51,3.17,-60.04,-59.8,-59.47,-39.96,-42.15,1.78,8.94,-8.87,3.11,7.64,-42.44,-10.9,-10.44,-12.34,1.18,-9.95,3.91,0.71,27.79,20.05,13.16,-13.18,5.0,-3.08,42.0,0.0,0.0,0.0,1651.0,2139.0,1856.0,1727.0,2119.0,2192.0,1604.0,0.0,0.0,0.0,1.0,0.0,0.0,1.0,1.0,0.0,1.0

RSI example file contains:
2AEE8631, C6F0202, 1604, 8, HK Frame, 1651, 2139, 1856, 1727, 2119, 2192, -146.53, -146.51,  7.63,  -29.93,  -29.95,  -10.24, -11.04, 1.85, 8.20, -9.02, -0.07, 3.91, 0.66, 27.84, 19.94
                                      MTR1  MTR2  MTR3  MTR4  MTR5  MTR6, SDD2Bias SDD1Bias  ArmRes  SDD1TMP SDD2TMP  ?       ?       ?     ?     ?      ?      FVMON FIMON HVMON  HIMON

2AEE8651, C6F0202, 1604, 8, HK Frame, 1651, 2139, 1856, 1727, 2119, 2192, -146.51, -146.48, 7.65, -30.00, -29.95, -10.56, -11.07, 1.88, 8.23, -9.25, 0.21, 3.93, 0.68, 27.80, 20.05
2AEE865D, C6F0202, 1604, 8, HK Frame, 1651, 2139, 1856, 1727, 2119, 2192, -146.51, -146.50, 7.64, -30.01, -30.00, -10.82, -10.69, 1.91, 8.43, -9.31, 0.47, 3.91, 0.71, 27.79, 20.05

SDF-Raw:
2022-302T00:22:08 : 1604 hk fcnt:10329 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEE8631 - Power Control: 0x99800668
2022-302T00:22:08 : 1604 hk E-Box - Analog FPGA Tmp:  21.06 C  | Mot Volt pos:   4.93 V |   DSPC plus 1.8V:   1.78 V
2022-302T00:22:08 : 1604 hk E-Box - Chassis Top Tmp:  14.17 C  | Mot Volt neg:  -4.91 V |        DSPC Pos :   8.94 V
2022-302T00:22:08 : 1604 hk E-Box - Chassis Bot Tmp:  14.13 C  |    SDD2 Bias:-146.54 V |        DSPC Neg :  -8.84 V
2022-302T00:22:08 : 1604 hk E-Box -     AFT Low Cal: -43.63    |    SDD1 Bias:-146.52 V |      PRT Current:   3.11 Amps
2022-302T00:22:08 : 1604 hk E-Box -    AFT High Cal:  41.97    |    PIXL 3.3 :   3.17 V |   Arm Resistance:   7.63 Ohm
2022-302T00:22:08 : 1604 hk SenHd -       SDD 1 Tmp: -29.93 C  |  Bipod 1 Tmp: -60.29 C |         FLIE Tmp: -42.47 C
2022-302T00:22:08 : 1604 hk SenHd -       SDD 2 Tmp: -29.95 C  |  Bipod 2 Tmp: -59.40 C |        TEC 1 Tmp: -10.31 C
2022-302T00:22:08 : 1604 hk SenHd -         AFE Tmp:   7.67 C  |  Bipod 3 Tmp: -59.40 C |        TEC 2 Tmp: -10.79 C
2022-302T00:22:08 : 1604 hk SenHd -        LVCM Tmp:  -9.42 C  |    Cover Tmp: -39.70 C |   Xray Bench Tmp: -12.14 C
2022-302T00:22:08 : 1604 hk SenHd -        HVMM Tmp:  -0.77 C  |      HOP Tmp: -42.41 C | Yellow Piece Tmp:   1.11 C
2022-302T00:22:08 : 1604 hk SenHd -                            |                        |          MCC Tmp: -10.31 C
2022-302T00:22:08 : 1604 hk  HVPS -           FVMON:   3.91 v  |  13 volt pos:  13.15 V |       Valid Cmds:    42
2022-302T00:22:08 : 1604 hk  HVPS -           FIMON:   0.66 v  |  13 volt neg: -13.15 V |        CRF Retry:     0
2022-302T00:22:08 : 1604 hk  HVPS -           HVMON:  27.84 Kv |   5 volt pos:   5.00 V |        SDF Retry:     0
2022-302T00:22:08 : 1604 hk  HVPS -           HIMON:  19.94 ua |     LVCM Tmp:  -3.31 C |    Cmds Rejected:     0
2022-302T00:22:08 : 1604 hk   Motor Pos:   1651   2139   1856   1727   2119   2192   Cvr:   0 -- Motor Sense: 0x0464
2022-302T00:22:08 : 1604 hk   Motor Sen:   vvvv   ^^^^   vvvv   vvvv   ^^^^   ^^^^   Cover is Partially Open
2022-302T00:22:08 : 1604 hk Breadcrumbs: 0xf80039f1 0x00238d00 0x0cd4000d 0x25000036 0x000d816f 0x4ae4f2ca
2022-302T00:22:08 : 1604 hk raw -------->> 28592288 21B121B0 1AA72514 133DEA6F E583E582 0C6406F1 22ECDB37 0C282A0B
2022-302T00:22:08 : 1604 hk raw -------->> 095E0960 20F71EDE 1FF518A9 18AD18B3 1B221AD3 1AD71EB8 1E9F1E82 20311EB0
2022-302T00:22:08 : 1604 hk raw -------->> 0C81021E 0D7F0CC2 0E06F1FA 055408A2 002A0000 00000000 0000DEAD DEADDEAD
2022-302T00:22:08 : 1604 hk raw -------->> 0673085B 074006BF 08470890 00000000 04640000 DEADDEAD DEADDEAD 190425F4
2022-302T00:22:08 : 1604 hk raw -------->> 2AEE8631 99800668 F80039F1 00238D00 0CD4000D 25000036 000D816F 4AE4F2CA


2022-302T00:22:08 : 1604 hk raw -------->> [FCNT?]DSPC- [8625][8624] [6823][LVCMTmp?] [MTRPos][60015] [58755][58754] [PIXLV][DSPC1.8] [DSPC+][56119] [PRTAmp][10763]
2022-302T00:22:08 : 1604 hk raw -------->> [2398][2400] [8439][7902] [8181][6313]     [6317][6323]    [6946][6867]   [6871][7864]     [7839][7810]   [8241][7856]
2022-302T00:22:08 : 1604 hk raw -------->> [3201][542]  [3455][3266] [3509][61946]    [1364][2210]    VCMD0000       00000000         0000DEAD       DEADDEAD
2022-302T00:22:08 : 1604 hk raw -------->> MTRPos1-6--------------------------------> 00000000        MTRSens0000    DEADDEAD         DEADDEAD       [6404][9716]
2022-302T00:22:08 : 1604 hk raw -------->> HKTIME       PWRCTRL      BREADCRUMBS----------------------------------------------------------------------------------->

2022-302T00:22:40 : 1604 hk fcnt:10333 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEE8651 - Power Control: 0x99800668
2022-302T00:22:40 : 1604 hk E-Box - Analog FPGA Tmp:  21.00 C  | Mot Volt pos:   4.93 V |   DSPC plus 1.8V:   1.78 V
2022-302T00:22:40 : 1604 hk E-Box - Chassis Top Tmp:  14.10 C  | Mot Volt neg:  -4.93 V |        DSPC Pos :   8.94 V
2022-302T00:22:40 : 1604 hk E-Box - Chassis Bot Tmp:  14.13 C  |    SDD2 Bias:-146.52 V |        DSPC Neg :  -8.85 V
2022-302T00:22:40 : 1604 hk E-Box -     AFT Low Cal: -43.66    |    SDD1 Bias:-146.49 V |      PRT Current:   3.12 Amps
2022-302T00:22:40 : 1604 hk E-Box -    AFT High Cal:  41.94    |    PIXL 3.3 :   3.17 V |   Arm Resistance:   7.65 Ohm
2022-302T00:22:40 : 1604 hk SenHd -       SDD 1 Tmp: -30.00 C  |  Bipod 1 Tmp: -60.29 C |         FLIE Tmp: -42.41 C
2022-302T00:22:40 : 1604 hk SenHd -       SDD 2 Tmp: -29.95 C  |  Bipod 2 Tmp: -59.21 C |        TEC 1 Tmp: -10.64 C
2022-302T00:22:40 : 1604 hk SenHd -         AFE Tmp:   7.70 C  |  Bipod 3 Tmp: -59.65 C |        TEC 2 Tmp: -10.82 C
2022-302T00:22:40 : 1604 hk SenHd -        LVCM Tmp:  -9.65 C  |    Cover Tmp: -39.74 C |   Xray Bench Tmp: -12.08 C
2022-302T00:22:40 : 1604 hk SenHd -        HVMM Tmp:  -0.48 C  |      HOP Tmp: -42.50 C | Yellow Piece Tmp:   1.15 C
2022-302T00:22:40 : 1604 hk SenHd -                            |                        |          MCC Tmp: -10.24 C
2022-302T00:22:40 : 1604 hk  HVPS -           FVMON:   3.93 v  |  13 volt pos:  13.15 V |       Valid Cmds:    42
2022-302T00:22:40 : 1604 hk  HVPS -           FIMON:   0.68 v  |  13 volt neg: -13.20 V |        CRF Retry:     0
2022-302T00:22:40 : 1604 hk  HVPS -           HVMON:  27.80 Kv |   5 volt pos:   5.00 V |        SDF Retry:     0
2022-302T00:22:40 : 1604 hk  HVPS -           HIMON:  20.05 ua |     LVCM Tmp:  -3.17 C |    Cmds Rejected:     0
2022-302T00:22:40 : 1604 hk   Motor Pos:   1651   2139   1856   1727   2119   2192   Cvr:   0 -- Motor Sense: 0x0464
2022-302T00:22:40 : 1604 hk   Motor Sen:   vvvv   ^^^^   vvvv   vvvv   ^^^^   ^^^^   Cover is Partially Open
2022-302T00:22:40 : 1604 hk Breadcrumbs: 0xf80029f1 0x0028f800 0x0c64000d 0x25000036 0x000d816f 0x4ae4f2ca
2022-302T00:22:40 : 1604 hk raw -------->> 285D2286 21AF21B0 1AA62513 133DEA69 E582E580 0C6606F0 22ECDB35 0C2D2A0D
2022-302T00:22:40 : 1604 hk raw -------->> 09680960 20F81ED7 1FFE18A9 18B318AB 1B211AD0 1AD91EAE 1E9E1E84 20321EB2
2022-302T00:22:40 : 1604 hk raw -------->> 0C8F022A 0D7A0CD5 0E05F1ED 0554089B 002A0000 00000000 0000DEAD DEADDEAD
2022-302T00:22:40 : 1604 hk raw -------->> 0673085B 074006BF 08470890 00000000 04640000 DEADDEAD DEADDEAD 190425F4
2022-302T00:22:40 : 1604 hk raw -------->> 2AEE8651 99800668 F80029F1 0028F800 0C64000D 25000036 000D816F 4AE4F2CA

2022-302T00:22:52 : 1604 hk fcnt:10335 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEE865D - Power Control: 0x99800668
2022-302T00:22:52 : 1604 hk E-Box - Analog FPGA Tmp:  20.97 C  | Mot Volt pos:   4.92 V |   DSPC plus 1.8V:   1.78 V
2022-302T00:22:52 : 1604 hk E-Box - Chassis Top Tmp:  14.13 C  | Mot Volt neg:  -4.90 V |        DSPC Pos :   8.94 V
2022-302T00:22:52 : 1604 hk E-Box - Chassis Bot Tmp:  14.10 C  |    SDD2 Bias:-146.52 V |        DSPC Neg :  -8.87 V
2022-302T00:22:52 : 1604 hk E-Box -     AFT Low Cal: -43.63    |    SDD1 Bias:-146.51 V |      PRT Current:   3.11 Amps
2022-302T00:22:52 : 1604 hk E-Box -    AFT High Cal:  41.97    |    PIXL 3.3 :   3.17 V |   Arm Resistance:   7.64 Ohm
2022-302T00:22:52 : 1604 hk SenHd -       SDD 1 Tmp: -30.01 C  |  Bipod 1 Tmp: -60.04 C |         FLIE Tmp: -42.44 C
2022-302T00:22:52 : 1604 hk SenHd -       SDD 2 Tmp: -30.00 C  |  Bipod 2 Tmp: -59.80 C |        TEC 1 Tmp: -10.90 C
2022-302T00:22:52 : 1604 hk SenHd -         AFE Tmp:   7.90 C  |  Bipod 3 Tmp: -59.47 C |        TEC 2 Tmp: -10.44 C
2022-302T00:22:52 : 1604 hk SenHd -        LVCM Tmp:  -9.71 C  |    Cover Tmp: -39.96 C |   Xray Bench Tmp: -12.34 C
2022-302T00:22:52 : 1604 hk SenHd -        HVMM Tmp:  -0.22 C  |      HOP Tmp: -42.15 C | Yellow Piece Tmp:   1.18 C
2022-302T00:22:52 : 1604 hk SenHd -                            |                        |          MCC Tmp:  -9.95 C
2022-302T00:22:52 : 1604 hk  HVPS -           FVMON:   3.91 v  |  13 volt pos:  13.16 V |       Valid Cmds:    42
2022-302T00:22:52 : 1604 hk  HVPS -           FIMON:   0.71 v  |  13 volt neg: -13.18 V |        CRF Retry:     0
2022-302T00:22:52 : 1604 hk  HVPS -           HVMON:  27.79 Kv |   5 volt pos:   5.00 V |        SDF Retry:     0
2022-302T00:22:52 : 1604 hk  HVPS -           HIMON:  20.05 ua |     LVCM Tmp:  -3.08 C |    Cmds Rejected:     0
2022-302T00:22:52 : 1604 hk   Motor Pos:   1651   2139   1856   1727   2119   2192   Cvr:   0 -- Motor Sense: 0x0464
2022-302T00:22:52 : 1604 hk   Motor Sen:   vvvv   ^^^^   vvvv   vvvv   ^^^^   ^^^^   Cover is Partially Open
2022-302T00:22:52 : 1604 hk Breadcrumbs: 0xf80039f1 0x00238d00 0x0c04000d 0x25000035 0x000d816f 0x4ae4f2ca
2022-302T00:22:52 : 1604 hk raw -------->> 285F2285 21B021AF 1AA72514 1333EA78 E582E581 0C6406F0 22ECDB17 0C272A0C
2022-302T00:22:52 : 1604 hk raw -------->> 09690968 20FE1ED5 200618B1 18A018B1 1B1A1ADB 1AD81EA6 1EAA1E7C 20331EBB
2022-302T00:22:52 : 1604 hk raw -------->> 0C850248 0D780CD4 0E0AF1F3 05540897 002A0000 00000000 0000DEAD DEADDEAD
2022-302T00:22:52 : 1604 hk raw -------->> 0673085B 074006BF 08470890 00000000 04640000 DEADDEAD DEADDEAD 190425F4
2022-302T00:22:52 : 1604 hk raw -------->> 2AEE865D 99800668 F80039F1 00238D00 0C04000D 25000035 000D816F 4AE4F2CA
*/

// Expects:
// fcnt:10329 - FPGAVersion: 0x190425F4 - HK Time: 0x2AEE8631 - Power Control: 0x99800668
// hkLines = [
// 2022-302T00:22:08 : 1604 hk E-Box - Analog FPGA Tmp:  21.06 C  | Mot Volt pos:   4.93 V |   DSPC plus 1.8V:   1.78 V
// 2022-302T00:22:08 : 1604 hk E-Box - Chassis Top Tmp:  14.17 C  | Mot Volt neg:  -4.91 V |        DSPC Pos :   8.94 V
// 2022-302T00:22:08 : 1604 hk E-Box - Chassis Bot Tmp:  14.13 C  |    SDD2 Bias:-146.54 V |        DSPC Neg :  -8.84 V
// 2022-302T00:22:08 : 1604 hk E-Box -     AFT Low Cal: -43.63    |    SDD1 Bias:-146.52 V |      PRT Current:   3.11 Amps
// 2022-302T00:22:08 : 1604 hk E-Box -    AFT High Cal:  41.97    |    PIXL 3.3 :   3.17 V |   Arm Resistance:   7.63 Ohm
// 2022-302T00:22:08 : 1604 hk SenHd -       SDD 1 Tmp: -29.93 C  |  Bipod 1 Tmp: -60.29 C |         FLIE Tmp: -42.47 C
// 2022-302T00:22:08 : 1604 hk SenHd -       SDD 2 Tmp: -29.95 C  |  Bipod 2 Tmp: -59.40 C |        TEC 1 Tmp: -10.31 C
// 2022-302T00:22:08 : 1604 hk SenHd -         AFE Tmp:   7.67 C  |  Bipod 3 Tmp: -59.40 C |        TEC 2 Tmp: -10.79 C
// 2022-302T00:22:08 : 1604 hk SenHd -        LVCM Tmp:  -9.42 C  |    Cover Tmp: -39.70 C |   Xray Bench Tmp: -12.14 C
// 2022-302T00:22:08 : 1604 hk SenHd -        HVMM Tmp:  -0.77 C  |      HOP Tmp: -42.41 C | Yellow Piece Tmp:   1.11 C
// 2022-302T00:22:08 : 1604 hk SenHd -                            |                        |          MCC Tmp: -10.31 C
// 2022-302T00:22:08 : 1604 hk  HVPS -           FVMON:   3.91 v  |  13 volt pos:  13.15 V |       Valid Cmds:    42
// 2022-302T00:22:08 : 1604 hk  HVPS -           FIMON:   0.66 v  |  13 volt neg: -13.15 V |        CRF Retry:     0
// 2022-302T00:22:08 : 1604 hk  HVPS -           HVMON:  27.84 Kv |   5 volt pos:   5.00 V |        SDF Retry:     0
// 2022-302T00:22:08 : 1604 hk  HVPS -           HIMON:  19.94 ua |     LVCM Tmp:  -3.31 C |    Cmds Rejected:     0
// 2022-302T00:22:08 : 1604 hk   Motor Pos:   1651   2139   1856   1727   2119   2192   Cvr:   0 -- Motor Sense: 0x0464
// 2022-302T00:22:08 : 1604 hk   Motor Sen:   vvvv   ^^^^   vvvv   vvvv   ^^^^   ^^^^   Cover is Partially Open
// 2022-302T00:22:08 : 1604 hk Breadcrumbs: 0xf80039f1 0x00238d00 0x0cd4000d 0x25000036 0x000d816f 0x4ae4f2ca
// 2022-302T00:22:08 : 1604 hk raw -------->> 28592288 21B121B0 1AA72514 133DEA6F E583E582 0C6406F1 22ECDB37 0C282A0B
// 2022-302T00:22:08 : 1604 hk raw -------->> 095E0960 20F71EDE 1FF518A9 18AD18B3 1B221AD3 1AD71EB8 1E9F1E82 20311EB0
// 2022-302T00:22:08 : 1604 hk raw -------->> 0C81021E 0D7F0CC2 0E06F1FA 055408A2 002A0000 00000000 0000DEAD DEADDEAD
// 2022-302T00:22:08 : 1604 hk raw -------->> 0673085B 074006BF 08470890 00000000 04640000 DEADDEAD DEADDEAD 190425F4
// 2022-302T00:22:08 : 1604 hk raw -------->> 2AEE8631 99800668 F80039F1 00238D00 0CD4000D 25000036 000D816F 4AE4F2CA
// ]
func processHousekeeping(lineNo int, lineData string, lines []string, sclk string, rtt int64, pmc int) (int64, string, string, error) {
	if len(lines) != 23 {
		return 0, "", "", fmt.Errorf("hk line count invalid on line %v", lineNo)
	}

	hktime, _, _, err := readNumBetween(lineData, "HK Time: 0x", " ", read_int_hex)
	if err != nil || hktime <= 0 {
		return 0, "", "", fmt.Errorf("hk start didn't contain hk time on line %v", lineNo)
	}

	fcnt, _, _, err := readNumBetween(lineData, "fcnt:", " ", read_int)
	if err != nil || hktime <= 0 {
		return 0, "", "", fmt.Errorf("hk start didn't contain fcnt on line %v", lineNo)
	}

	// Snip all lines so they start after mcc_trn
	tok := fmt.Sprintf("%v hk", pmc)
	for c := 0; c < 23; c++ {
		pos := strings.Index(lines[c], tok)
		if pos < 0 {
			return 0, "", "", fmt.Errorf("%v not found on line %v", tok, lineNo)
		}

		lines[c] = strings.Trim(lines[c][pos+len(tok):], " ")
	}

	var ok bool
	motorPos := []int{}
	fVal := []float32{}

	tok, lines[15], ok = takeToken(lines[15], ":")
	if !ok || tok != "Motor Pos" {
		return 0, "", "", fmt.Errorf("Expected Motor Pos, got %v on line %v", tok, lineNo)
	}

	for c := 0; c < 6; c++ {
		var p int64
		p, lines[15], err = readInt(lines[15])
		if err != nil {
			return 0, "", "", fmt.Errorf("Failed to read Motor Pos %v on line %v", c, lineNo)
		}
		motorPos = append(motorPos, int(p))
	}

	// Read the rest as floats
	var pos int
	var f float32

	names := []string{"SDD2 Bias:", "SDD1 Bias:", "Arm Resistance:", "SDD 1 Tmp:", "SDD 2 Tmp:", "FVMON:", "FIMON:", "HVMON:", "HIMON:"}

	lineOffset := 2
	for c := 0; c < len(names); c++ {
		_, f, pos, err = readNumBetween(lines[c+lineOffset], names[c], " ", read_float)
		if err != nil {
			return 0, "", "", err
		}
		if pos < 0 {
			return 0, "", "", fmt.Errorf("Missing value: %v", names[c])
		}
		fVal = append(fVal, f)

		if c == 4 {
			lineOffset = 6
		}
	}

	// Outputs:
	// 2AEE898E, C6F0202, 1658, 8, HK Frame, 1957, 1967, 2040, 1958, 1966, 2098, -146.50, -146.47, 7.64, -30.04, -30.02, -10.47, -11.04, 2.17, 8.87, -8.83, -0.04, 3.92, 0.70, 27.79, 20.05
	hk := fmt.Sprintf("%v, %X, %v, 8, HK Frame, %d, %d, %d, %d, %d, %d, %v, %v, %v, %v, %v, -1, -1, -1, -1, -1, -1, %v, %v, %v, %v\n",
		makeWriteSCLK(sclk), rtt, pmc,
		motorPos[0], motorPos[1], motorPos[2], motorPos[3], motorPos[4], motorPos[5],
		fVal[0], fVal[1], fVal[2], fVal[3], fVal[4], fVal[5], fVal[6], fVal[7], fVal[8])

	// We also output housekeeping data in a different "RSI" format thats compatible with the ones output by the pipeline for PIXLISE to read actual housekeeping
	// values from. This differs from the above, and doesn't have all the columns in the "real" files but PIXLISE gets a lot of what it needs this way already. If
	// specific data is required, we'll have to add it here

	// DataDrive RSI format has table headers:
	// HK Frame
	// SCLK,PMC,hk_fcnt,f_pixl_analog_fpga,f_pixl_chassis_top,f_pixl_chassis_bottom,f_pixl_aft_low_cal,f_pixl_aft_high_cal,f_pixl_motor_v_plus,f_pixl_motor_v_minus,f_pixl_sdd_1,f_pixl_sdd_2,f_pixl_3_3_volt,f_pixl_1_8_volt,f_pixl_dspc_v_plus,f_pixl_dspc_v_minus,f_pixl_prt_curr,f_pixl_arm_resist,f_head_sdd_1,f_head_sdd_2,f_head_afe,f_head_lvcm,f_head_hvmm,f_head_bipod1,f_head_bipod2,f_head_bipod3,f_head_cover,f_head_hop,f_head_flie,f_head_tec1,f_head_tec2,f_head_xray,f_head_yellow_piece,f_head_mcc,f_hvps_fvmon,f_hvps_fimon,f_hvps_hvmon,f_hvps_himon,f_hvps_13v_plus,f_hvps_13v_minus,f_hvps_5v_plus,f_hvps_lvcm,i_valid_cmds,i_crf_retry,i_sdf_retry,i_rejected_cmds,i_hk_side,i_motor_1,i_motor_2,i_motor_3,i_motor_4,i_motor_5,i_motor_6,i_motor_cover,i_hes_sense,i_flash_status,u_hk_version,u_hk_time,u_hk_power,u_fsw_0,u_fsw_1,u_fsw_2,u_fsw_3,u_fsw_4,u_fsw_5,f_pixl_analog_fpga_conv,f_pixl_chassis_top_conv,f_pixl_chassis_bottom_conv,f_pixl_aft_low_cal_conv,f_pixl_aft_high_cal_conv,f_pixl_motor_v_plus_conv,f_pixl_motor_v_minus_conv,f_pixl_sdd_1_conv,f_pixl_sdd_2_conv,f_pixl_3_3_volt_conv,f_pixl_1_8_volt_conv,f_pixl_dspc_v_plus_conv,f_pixl_dspc_v_minus_conv,f_pixl_prt_curr_conv,f_pixl_arm_resist_conv,f_head_sdd_1_conv,f_head_sdd_2_conv,f_head_afe_conv,f_head_lvcm_conv,f_head_hvmm_conv,f_head_bipod1_conv,f_head_bipod2_conv,f_head_bipod3_conv,f_head_cover_conv,f_head_hop_conv,f_head_flie_conv,f_head_tec1_conv,f_head_tec2_conv,f_head_xray_conv,f_head_yellow_piece_conv,f_head_mcc_conv,f_hvps_fvmon_conv,f_hvps_fimon_conv,f_hvps_hvmon_conv,f_hvps_himon_conv,f_hvps_13v_plus_conv,f_hvps_13v_minus_conv,f_hvps_5v_plus_conv,f_hvps_lvcm_conv,i_valid_cmds_conv,i_crf_retry_conv,i_sdf_retry_conv,i_rejected_cmds_conv,i_hk_side_conv,i_motor_1_conv,i_motor_2_conv,i_motor_3_conv,i_motor_4_conv,i_motor_5_conv,i_motor_6_conv,i_motor_cover_conv,i_hes_sense_conv,i_flash_status_conv,RTT
	// 720274993,1604,10329,8840,8625,8624,6823,9492,4925,60015,58755,58754,3172,1777,8940,56119,3112,10763,2398,2400,8439,7902,8181,6313,6317,6323,6946,6867,6871,7864,7839,7810,8241,7856,3201,542,3455,3266,3590,61946,1364,2210,42,0,0,0,0,1651,2139,1856,1727,2119,2192,0,1124,0,0x190425F4,720274993,0x99800668,0xF80039F1,0x00238D00,0x0CD4000D,0x25000036,0x000D816F,0x4AE4F2CA,21.06158,14.166139999999999,14.13406,-43.62744,41.97247,4.925,-4.9094,-146.53015,-146.51488999999998,3.172,1.777,8.94,-8.83681,3.112,7.63,-29.934690000000003,-29.948140000000002,8.20074,-9.02187,-0.07379,-59.9841,-59.855819999999994,-59.66339,-39.6826,-42.21628,-42.08798,-10.24059,-11.04239,-11.97247,1.8505200000000002,-10.497160000000001,3.90843,0.66178,27.84249,19.93895,13.150179999999999,-13.150179999999999,4.99634,-3.31345,42,0,0,0,0,1651,2139,1856,1727,2119,2192,0,1124,0,208601602

	// We output:
	// SCLK,PMC,hk_fcnt,f_pixl_sdd_1_conv,f_pixl_sdd_2_conv,f_pixl_arm_resist_conv,f_head_sdd_1_conv,f_head_sdd_2_conv,f_hvps_fvmon_conv,f_hvps_fimon_conv,f_hvps_hvmon_conv,f_hvps_himon_conv,i_motor_1_conv,i_motor_2_conv,i_motor_3_conv,i_motor_4_conv,i_motor_5_conv,i_motor_6_conv

	iSCLK, err := makeWriteSCLInt(sclk)
	if err != nil {
		return 0, "", "", fmt.Errorf("hk failed to parse SCLK on line %v: %v", lineNo, err)
	}

	hk2 := fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v, %v\n",
		iSCLK, pmc, fcnt, fVal[1], fVal[0], fVal[2], fVal[3], fVal[4], fVal[5], fVal[6], fVal[7], fVal[8],
		motorPos[0], motorPos[1], motorPos[2], motorPos[3], motorPos[4], motorPos[5])

	return hktime, hk, hk2, nil
}
