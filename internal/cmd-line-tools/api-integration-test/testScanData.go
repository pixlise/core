package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testScanData(apiHost string, groupDepth int) {
	scanId := seedDBScanData()
	// Prepend the special bit required for ownership table scan storage
	scanId = "scan_" + scanId

	// Dataset - bad dataset id
	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)
	userId := testScanDataBadId(apiHost, "Pseudo: ")

	noAccessTest := func(apiHost string) {
		// No viewers or editors on the item
		seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)
		// Clear user groups
		seedDBUserGroups(nil)

		// Run test expecting to get a no permission error
		testScanDataNoPermission(apiHost)
	}

	accessTest := func(apiHost string, comment string, ownershipViewers *protos.UserGroupList, ownershipEditors *protos.UserGroupList, userGroups []*protos.UserGroupDB) {
		// Set the viewers and editors
		seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, ownershipViewers, ownershipEditors)
		// Set user groups created
		seedDBUserGroups(userGroups)

		// Run test expecting to get
		testScanDataHasPermission(apiHost, comment)
	}

	wstestlib.RunFullAccessTest(apiHost, userId, groupDepth, noAccessTest, accessTest)
}

const scanWaitTime = 60 * 1000 * 10

func seedDBScanData() string {
	scanItem := protos.ScanItem{
		Id:    "048300551",
		Title: "Naltsos",
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			{
				DataType: protos.ScanDataType_SD_XRF,
				Count:    133,
			},
			{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    4,
			},
		},
		Instrument:       protos.ScanInstrument_PIXL_FM,
		InstrumentConfig: "PIXL",
		TimestampUnixSec: 1646262426,
		Meta: map[string]string{
			"Sol":      "0125",
			"DriveId":  "1712",
			"TargetId": "?",
			"Target":   "",
			"SiteId":   "4",
			"Site":     "",
			"RTT":      "048300551",
			"SCLK":     "678031418",
		},
		ContentCounts: map[string]int32{
			"BulkSpectra":       2,
			"MaxSpectra":        2,
			"PseudoIntensities": 121,
			"NormalSpectra":     242,
			"DwellSpectra":      0,
		},
		CreatorUserId: "",
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.ScansName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, &scanItem)
	if err != nil {
		log.Fatalln(err)
	}

	return scanItem.Id
}

func seedDBOwnership(objectId string, objectType protos.ObjectType, viewers *protos.UserGroupList, editors *protos.UserGroupList) {
	// Insert a scan item into DB for this
	scanOwnerItem := protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatorUserId:  "",
		CreatedUnixSec: 1646262426,
	}

	if viewers != nil {
		scanOwnerItem.Viewers = viewers
	}

	if editors != nil {
		scanOwnerItem.Editors = editors
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.OwnershipName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, &scanOwnerItem)
	if err != nil {
		log.Fatalln(err)
	}
}

func seedDBUserGroups(groups []*protos.UserGroupDB) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.UserGroupsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	if len(groups) > 0 {
		items := []interface{}{}
		for _, g := range groups {
			items = append(items, g)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func testScanDataBadId(apiHost string, actionMsg string) string {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	u1.AddSendReqAction(actionMsg+" (not found)",
		`{"pseudoIntensityReq":{"scanId": "non-existant-scan"}}`,
		`{"msgId":1, "status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"pseudoIntensityResp":{}
		}`,
	)

	u1.AddSendReqAction(actionMsg+" (not found)",
		`{"spectrumReq":{"scanId": "non-existant-scan"}}`,
		`{"msgId":2, "status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"spectrumResp":{}
		}`,
	)

	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
	return u1.GetUserId()
}

func testScanDataNoPermission(apiHost string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Using Naltsos: 048300551
	// First scan from Mars, should be in all environments, total size is only 800kb, 121 PMCs
	// NOTES: - Intensity label order matters, should be returned as defined here
	//        - We only need to know that we get the right number of locations, and that
	//          an individual item has the right length of intensities...
	u1.AddSendReqAction("pseudo (expect no permission)",
		`{"pseudoIntensityReq":{"scanId": "048300551", "startingLocation": 100, "locationCount": 5}}`,
		`{
			"msgId": 1,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: 048300551",
			"pseudoIntensityResp": {}
		}`,
	)

	u1.AddSendReqAction("spectrum (expect no permission)",
		`{"spectrumReq":{"scanId": "048300551", "startingLocation": 128, "locationCount": 4}}`,
		`{
			"msgId": 2,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: 048300551",
			"spectrumResp": {}
		}`,
	)

	u1.AddSendReqAction("metaLabels (expect no permission)",
		`{"scanMetaLabelsReq":{"scanId": "048300551"}}`,
		`{"msgId":3,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: 048300551",
			"scanMetaLabelsResp":{}
		}`,
	)

	u1.AddSendReqAction("scanLocation (expect no permission)",
		`{"scanLocationReq":{"scanId": "048300551"}}`,
		`{"msgId":4,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: 048300551",
			"scanLocationResp":{}
		}`,
	)

	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
}

func testScanDataHasPermission(apiHost string, actionMsg string) {
	u1 := wstestlib.MakeScriptedTestUser(auth0Params)
	u1.AddConnectAction("Connect", &wstestlib.ConnectInfo{
		Host: apiHost,
		User: test1Username,
		Pass: test1Password,
	})

	// Using Naltsos: 048300551
	// First scan from Mars, should be in all environments, total size is only 800kb, 121 PMCs
	// NOTES: - Intensity label order matters, should be returned as defined here
	//        - We only need to know that we get the right number of locations, and that
	//          an individual item has the right length of intensities...
	u1.AddSendReqAction("Pseudo: "+actionMsg+" (should work)",
		`{"pseudoIntensityReq":{"scanId": "048300551", "startingLocation": 100, "locationCount": 5}}`,
		`{"msgId":1, "status": "WS_OK",
			"pseudoIntensityResp":{
				"intensityLabels": [
					"Na",
					"Mg",
					"Al",
					"Si",
					"P",
					"S",
					"Cl",
					"K",
					"Ca",
					"Ti",
					"Ce",
					"Cr",
					"Mn",
					"Fe",
					"Ni",
					"Ge",
					"As",
					"Zn",
					"Sr",
					"Y",
					"Zr",
					"Ba"
				],
				"data${LIST,MODE=CONTAINS,MINLENGTH=4}": [
					{
						"intensities${LIST,MODE=LENGTH,LENGTH=32}": []
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("Spectra: "+actionMsg+" (should work)",
		`{"spectrumReq":{"scanId": "048300551", "startingLocation": 128, "locationCount": 4}}`,
		`{"msgId":2, "status": "WS_OK",
			"spectrumResp":{
				"spectraPerLocation": [
					{
						"spectra": [
							{
								"detector": "A",
								"type": "SPECTRUM_NORMAL",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "3096",
								"meta": {
									"0": {"ivalue": 678034825},
									"119": {"fvalue": 13.879905},
									"120": {"fvalue": -18.5},
									"123": {"fvalue": 15},
									"124": {"fvalue": 7.862}
								}
							},
							{
								"detector": "B",
								"type": "SPECTRUM_NORMAL",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "2855",
								"meta": {
									"0": {"ivalue": 678034826},
									"119": {"fvalue": 13.893265},
									"120": {"fvalue": -22.4},
									"123": {"fvalue": 15},
									"124": {"fvalue": 7.881}
								}
							}
						]
					},
					{},
					{
						"spectra": [
							{
								"detector": "A",
								"type": "SPECTRUM_BULK",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "367901",
								"meta": {
									"119": {"fvalue": 1712.4017},
									"120": {"fvalue": -11.8},
									"123": {"fvalue": 1815},
									"124": {"fvalue": 7.9226},
									"125": {"svalue": "YY"},
									"126": {"svalue": "EMSA/MAS spectral data file"},
									"127": {"svalue": "2"},
									"128": {"svalue": "4096"},
									"129": {"svalue": "PIXL Flight Model"},
									"130": {"svalue": "XRF"},
									"131": {"svalue": "N/A"},
									"132": {"svalue": "TC202v2.0 PIXL"},
									"133": {"svalue": "eV"},
									"134": {"svalue": "-1.032"},
									"135": {"svalue": "COUNTS"}
								}
							},
							{
								"detector": "B",
								"type": "SPECTRUM_BULK",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "353738",
								"meta": {
									"119": {"fvalue": 1712.503},
									"120": {"fvalue": -13.2},
									"123": {"fvalue": 1815},
									"124": {"fvalue": 7.9273},
									"125": {"svalue": "YY"},
									"126": {"svalue": "EMSA/MAS spectral data file"},
									"127": {"svalue": "2"},
									"128": {"svalue": "4096"},
									"129": {"svalue": "PIXL Flight Model"},
									"130": {"svalue": "XRF"},
									"131": {"svalue": "N/A"},
									"132": {"svalue": "TC202v2.0 PIXL"},
									"133": {"svalue": "eV"},
									"134": {"svalue": "-1.032"},
									"135": {"svalue": "COUNTS"}
								}
							},
							{
								"detector": "A",
								"type": "SPECTRUM_MAX",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "7017",
								"meta": {
									"119": {"fvalue": 14.293016},
									"120": {"fvalue": -11.8},
									"123": {"fvalue": 15},
									"124": {"fvalue": 7.9226},
									"125": {"svalue": "YY"},
									"126": {"svalue": "EMSA/MAS spectral data file"},
									"127": {"svalue": "2"},
									"128": {"svalue": "4096"},
									"129": {"svalue": "PIXL Flight Model"},
									"130": {"svalue": "XRF"},
									"131": {"svalue": "N/A"},
									"132": {"svalue": "TC202v2.0 PIXL"},
									"133": {"svalue": "eV"},
									"134": {"svalue": "-1.032"},
									"135": {"svalue": "COUNTS"}
								}
							},
							{
								"detector": "B",
								"type": "SPECTRUM_MAX",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": "6786",
								"meta": {
									"119": {"fvalue": 14.27207},
									"120": {"fvalue": -13.2},
									"123": {"fvalue": 15},
									"124": {"fvalue": 7.9273},
									"125": {"svalue": "YY"},
									"126": {"svalue": "EMSA/MAS spectral data file"},
									"127": {"svalue": "2"},
									"128": {"svalue": "4096"},
									"129": {"svalue": "PIXL Flight Model"},
									"130": {"svalue": "XRF"},
									"131": {"svalue": "N/A"},
									"132": {"svalue": "TC202v2.0 PIXL"},
									"133": {"svalue": "eV"},
									"134": {"svalue": "-1.032"},
									"135": {"svalue": "COUNTS"}
								}
							}
						]
					},
					{}
				]
			}
		}`,
	)

	u1.AddSendReqAction("MetaLabels: "+actionMsg+" (should work)",
		`{"scanMetaLabelsReq":{"scanId": "048300551"}}`,
		`{"msgId":3, "status": "WS_OK",
			"scanMetaLabelsResp":{
				"metaLabels": [
				"SCLK",
				"hk_fcnt",
				"f_pixl_analog_fpga",
				"f_pixl_chassis_top",
				"f_pixl_chassis_bottom",
				"f_pixl_aft_low_cal",
				"f_pixl_aft_high_cal",
				"f_pixl_motor_v_plus",
				"f_pixl_motor_v_minus",
				"f_pixl_sdd_1",
				"f_pixl_sdd_2",
				"f_pixl_3_3_volt",
				"f_pixl_1_8_volt",
				"f_pixl_dspc_v_plus",
				"f_pixl_dspc_v_minus",
				"f_pixl_prt_curr",
				"f_pixl_arm_resist",
				"f_head_sdd_1",
				"f_head_sdd_2",
				"f_head_afe",
				"f_head_lvcm",
				"f_head_hvmm",
				"f_head_bipod1",
				"f_head_bipod2",
				"f_head_bipod3",
				"f_head_cover",
				"f_head_hop",
				"f_head_flie",
				"f_head_tec1",
				"f_head_tec2",
				"f_head_xray",
				"f_head_yellow_piece",
				"f_head_mcc",
				"f_hvps_fvmon",
				"f_hvps_fimon",
				"f_hvps_hvmon",
				"f_hvps_himon",
				"f_hvps_13v_plus",
				"f_hvps_13v_minus",
				"f_hvps_5v_plus",
				"f_hvps_lvcm",
				"i_valid_cmds",
				"i_crf_retry",
				"i_sdf_retry",
				"i_rejected_cmds",
				"i_hk_side",
				"i_motor_1",
				"i_motor_2",
				"i_motor_3",
				"i_motor_4",
				"i_motor_5",
				"i_motor_6",
				"i_motor_cover",
				"i_hes_sense",
				"i_flash_status",
				"u_hk_version",
				"u_hk_time",
				"u_hk_power",
				"u_fsw_0",
				"u_fsw_1",
				"u_fsw_2",
				"u_fsw_3",
				"u_fsw_4",
				"u_fsw_5",
				"f_pixl_analog_fpga_conv",
				"f_pixl_chassis_top_conv",
				"f_pixl_chassis_bottom_conv",
				"f_pixl_aft_low_cal_conv",
				"f_pixl_aft_high_cal_conv",
				"f_pixl_motor_v_plus_conv",
				"f_pixl_motor_v_minus_conv",
				"f_pixl_sdd_1_conv",
				"f_pixl_sdd_2_conv",
				"f_pixl_3_3_volt_conv",
				"f_pixl_1_8_volt_conv",
				"f_pixl_dspc_v_plus_conv",
				"f_pixl_dspc_v_minus_conv",
				"f_pixl_prt_curr_conv",
				"f_pixl_arm_resist_conv",
				"f_head_sdd_1_conv",
				"f_head_sdd_2_conv",
				"f_head_afe_conv",
				"f_head_lvcm_conv",
				"f_head_hvmm_conv",
				"f_head_bipod1_conv",
				"f_head_bipod2_conv",
				"f_head_bipod3_conv",
				"f_head_cover_conv",
				"f_head_hop_conv",
				"f_head_flie_conv",
				"f_head_tec1_conv",
				"f_head_tec2_conv",
				"f_head_xray_conv",
				"f_head_yellow_piece_conv",
				"f_head_mcc_conv",
				"f_hvps_fvmon_conv",
				"f_hvps_fimon_conv",
				"f_hvps_hvmon_conv",
				"f_hvps_himon_conv",
				"f_hvps_13v_plus_conv",
				"f_hvps_13v_minus_conv",
				"f_hvps_5v_plus_conv",
				"f_hvps_lvcm_conv",
				"i_valid_cmds_conv",
				"i_crf_retry_conv",
				"i_sdf_retry_conv",
				"i_rejected_cmds_conv",
				"i_hk_side_conv",
				"i_motor_1_conv",
				"i_motor_2_conv",
				"i_motor_3_conv",
				"i_motor_4_conv",
				"i_motor_5_conv",
				"i_motor_6_conv",
				"i_motor_cover_conv",
				"i_hes_sense_conv",
				"i_flash_status_conv",
				"RTT",
				"DETECTOR_ID",
				"LIVETIME",
				"OFFSET",
				"PMC",
				"READTYPE",
				"REALTIME",
				"XPERCHAN",
				"DATATYPE",
				"FORMAT",
				"NCOLUMNS",
				"NPOINTS",
				"OWNER",
				"SIGNALTYPE",
				"TITLE",
				"VERSION",
				"XUNITS",
				"YP_TEMP",
				"YUNITS"
				]
			}
		}`,
	)

	u1.AddSendReqAction("scanLocation (should work)",
		`{"scanLocationReq":{"scanId": "048300551", "startingLocation": 128, "locationCount": 4}}`,
		`{"msgId":4, "status": "WS_OK",
			"scanLocationResp":{
				"locations${LIST,MODE=CONTAINS,LENGTH=4}": [
					{
						"id": 217,
						"meta": {
							"0": {"ivalue": 678034966},
							"1": {"ivalue": 1442},
							"2": {"ivalue": 9143},
							"3": {"ivalue": 8931},
							"4": {"ivalue": 8934},
							"5": {"ivalue": 6823},
							"6": {"ivalue": 9488},
							"7": {"ivalue": 4928},
							"8": {"ivalue": 59998},
							"9": {"ivalue": 58757},
							"10": {"ivalue": 58756},
							"11": {"ivalue": 3167},
							"12": {"ivalue": 1778},
							"13": {"ivalue": 8944},
							"14": {"ivalue": 56064},
							"15": {"ivalue": 3101},
							"16": {"ivalue": 11122},
							"17": {"ivalue": 2411},
							"18": {"ivalue": 2407},
							"19": {"ivalue": 8603},
							"20": {"ivalue": 8170},
							"21": {"ivalue": 8073},
							"22": {"ivalue": 7702},
							"23": {"ivalue": 7713},
							"24": {"ivalue": 7666},
							"25": {"ivalue": 7485},
							"26": {"ivalue": 7434},
							"27": {"ivalue": 7497},
							"28": {"ivalue": 8097},
							"29": {"ivalue": 8080},
							"30": {"ivalue": 8036},
							"31": {"ivalue": 8201},
							"32": {"ivalue": 8055},
							"33": {"ivalue": 3221},
							"34": {"ivalue": 595},
							"35": {"ivalue": 3449},
							"36": {"ivalue": 3293},
							"37": {"ivalue": 3590},
							"38": {"ivalue": 61931},
							"39": {"ivalue": 1365},
							"40": {"ivalue": 1802},
							"41": {"ivalue": 221},
							"42": {"ivalue": 0},
							"43": {"ivalue": 0},
							"44": {"ivalue": 0},
							"45": {"ivalue": 0},
							"46": {"ivalue": 1965},
							"47": {"ivalue": 1974},
							"48": {"ivalue": 2047},
							"49": {"ivalue": 1964},
							"50": {"ivalue": 1972},
							"51": {"ivalue": 2104},
							"52": {"ivalue": 864},
							"53": {"ivalue": 1088},
							"54": {"ivalue": 0},
							"55": {"svalue": "0x190425F4"},
							"56": {"ivalue": 678034966},
							"57": {"svalue": "0x99800668"},
							"58": {"svalue": "0xF800392C"},
							"59": {"svalue": "0x00238D00"},
							"60": {"svalue": "0x0C04000D"},
							"61": {"svalue": "0x25000001"},
							"62": {"svalue": "0x000C8AA3"},
							"63": {"svalue": "0x4AE4F2CA"},
							"64": {"fvalue": 30.7794},
							"65": {"fvalue": 23.9801},
							"66": {"fvalue": 24.0764},
							"67": {"fvalue": -43.6274},
							"68": {"fvalue": 41.8442},
							"69": {"fvalue": 4.928},
							"70": {"fvalue": -4.90153},
							"71": {"fvalue": -146.561},
							"72": {"fvalue": -146.545},
							"73": {"fvalue": 3.167},
							"74": {"fvalue": 1.778},
							"75": {"fvalue": 8.944},
							"76": {"fvalue": -8.86726},
							"77": {"fvalue": 3.101},
							"78": {"fvalue": 11.22},
							"79": {"fvalue": -30.0218},
							"80": {"fvalue": -29.9951},
							"81": {"fvalue": 13.4605},
							"82": {"fvalue": -0.426605},
							"83": {"fvalue": -3.53757},
							"84": {"fvalue": -15.4362},
							"85": {"fvalue": -15.0835},
							"86": {"fvalue": -16.5908},
							"87": {"fvalue": -22.3958},
							"88": {"fvalue": -24.0315},
							"89": {"fvalue": -22.011},
							"90": {"fvalue": -2.76785},
							"91": {"fvalue": -3.31305},
							"92": {"fvalue": -4.72421},
							"93": {"fvalue": 0.567627},
							"94": {"fvalue": -4.11487},
							"95": {"fvalue": 3.93285},
							"96": {"fvalue": 0.726496},
							"97": {"fvalue": 27.7941},
							"98": {"fvalue": 20.1038},
							"99": {"fvalue": 13.1502},
							"100": {"fvalue": -13.2051},
							"101": {"fvalue": 5},
							"102": {"fvalue": 5.35852},
							"103": {"ivalue": 221},
							"104": {"ivalue": 0},
							"105": {"ivalue": 0},
							"106": {"ivalue": 0},
							"107": {"ivalue": 0},
							"108": {"ivalue": 1965},
							"109": {"ivalue": 1974},
							"110": {"ivalue": 2047},
							"111": {"ivalue": 1964},
							"112": {"ivalue": 1972},
							"113": {"ivalue": 2104},
							"114": {"ivalue": 864},
							"115": {"ivalue": 1088},
							"116": {"ivalue": 0},
							"117": {"ivalue": 48300551}
						}
					}
				]
			}
		}`,
	)

	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
}