package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func testScanData(apiHost string, groupDepth int) {
	scanId := seedDBScanData(scan_Naltsos)

	// Prepend the special bit required for ownership table scan storage
	//scanId = "scan_" + scanId

	seedImages()
	seedImageLocations()
	// Seed the diffraction DB
	seedS3File(scanId+"-diffraction-db.bin", filepaths.GetScanFilePath(scanId, filepaths.DiffractionDBFileName), apiDatasetBucket)

	seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)

	// Dataset - bad dataset id
	userId := testScanDataBadId(apiHost, "Pseudo: ")

	noAccessTest := func(apiHost string) {
		// No viewers or editors on the item
		seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, nil, nil)
		// Clear user groups
		seedDBUserGroups(nil)

		// Run test expecting to get a no permission error
		testScanDataNoPermission(apiHost)
	}

	accessTest := func(apiHost string, comment string, ownershipViewers *protos.UserGroupList, ownershipEditors *protos.UserGroupList, userGroups []*protos.UserGroupDB, editAllowed bool) {
		// Set the viewers and editors
		seedDBOwnership(scanId, protos.ObjectType_OT_SCAN, ownershipViewers, ownershipEditors)
		// Set user groups created
		seedDBUserGroups(userGroups)

		// Run test expecting to get
		testScanDataHasPermission(apiHost, comment, editAllowed)
	}

	wstestlib.RunFullAccessTest(apiHost, userId, groupDepth, noAccessTest, accessTest)
}

const scanWaitTime = 60 * 1000 // why was this set to 10min initially? * 10

var scan_Naltsos = &protos.ScanItem{
	Id:    "048300551",
	Title: "",
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

var scan_Beaujeu = &protos.ScanItem{
	Id:    "052822532",
	Title: "",
	DataTypes: []*protos.ScanItem_ScanTypeCount{
		{
			DataType: protos.ScanDataType_SD_XRF,
			Count:    450,
		},
		{
			DataType: protos.ScanDataType_SD_IMAGE,
			Count:    4,
		},
	},
	Instrument:       protos.ScanInstrument_PIXL_FM,
	InstrumentConfig: "PIXL",
	TimestampUnixSec: 1663626388,
	Meta: map[string]string{
		"Sol":      "0138",
		"DriveId":  "1812",
		"TargetId": "?",
		"Target":   "",
		"SiteId":   "5",
		"Site":     "",
		"RTT":      "052822532",
		"SCLK":     "679215716",
	},
	ContentCounts: map[string]int32{
		"BulkSpectra":       2,
		"MaxSpectra":        2,
		"PseudoIntensities": 225,
		"NormalSpectra":     450,
		"DwellSpectra":      0,
	},
	CreatorUserId: "",
}

func seedDBScanData(scan *protos.ScanItem) string {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.ScansName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, &scan)
	if err != nil {
		log.Fatalln(err)
	}

	return scan.Id
}

func seedDBUserNotifications(userSettings map[string]*protos.UserNotificationSettings) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.UsersName)
	ctx := context.TODO()

	cursor, err := coll.Find(ctx, bson.D{}, options.Find())
	if err != nil {
		log.Fatalln(err)
	}

	users := []*protos.UserDBItem{}
	err = cursor.All(ctx, &users)
	if err != nil {
		return
	}

	usersToSave := []interface{}{}
	for _, user := range users {
		if setting := userSettings[user.Id]; setting != nil {
			user.NotificationSettings = setting
		}

		usersToSave = append(usersToSave, user)
	}

	// Clear the table
	err = coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	// Write the new ones out
	_, err = coll.InsertMany(ctx, usersToSave)
	if err != nil {
		log.Fatalln(err)
	}
}

func seedImages() {
	imgs := []interface{}{
		&protos.ScanImage{
			ImagePath:         "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
			Source:            1,
			Width:             752,
			Height:            580,
			FileSize:          240084,
			Purpose:           1,
			AssociatedScanIds: []string{"048300551"},
			OriginScanId:      "048300551",
			OriginImageURL:    "",
			//"matchinfo": null
		},
		&protos.ScanImage{
			ImagePath:         "048300551/PCW_0125_0678032223_000RCM_N00417120483005510093075J02.png",
			Source:            1,
			Width:             752,
			Height:            580,
			FileSize:          256736,
			Purpose:           1,
			AssociatedScanIds: []string{"048300551"},
			OriginScanId:      "048300551",
			OriginImageURL:    "",
		},
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.ImagesName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	res, err := coll.InsertMany(ctx, imgs)
	if err != nil {
		log.Fatalln(err)
	}

	if len(res.InsertedIDs) != len(imgs) {
		log.Fatalln("Failed to seed images")
	}
}

func seedImageLocations() {
	locs := []interface{}{
		&protos.ImageLocations{
			ImageName: "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
			LocationPerScan: []*protos.ImageLocationsForScan{
				{
					ScanId: "048300551",
					Locations: []*protos.Coordinate2D{
						nil,
						nil,
						nil,
						{
							I: 361.19134521484375,
							J: 293.5299072265625,
						},
						nil,
						{
							I: 65.51853942871094,
							J: 305.2160949707031,
						},
					},
				},
			},
		},
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.ImageBeamLocationsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	res, err := coll.InsertMany(ctx, locs)
	if err != nil {
		log.Fatalln(err)
	}

	if len(res.InsertedIDs) != len(locs) {
		log.Fatalln("Failed to seed image beam locations")
	}
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

func seedDBOwnershipMulti(ownerships []*protos.OwnershipItem, viewers *protos.UserGroupList, editors *protos.UserGroupList) {
	// Insert a scan item into DB for this
	ownershipIfcs := []interface{}{}
	for _, owner := range ownerships {
		if viewers != nil {
			owner.Viewers = viewers
		}

		if editors != nil {
			owner.Editors = editors
		}

		ownershipIfcs = append(ownershipIfcs, owner)
	}

	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.OwnershipName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertMany(ctx, ownershipIfcs)
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

	u1.AddSendReqAction("non-existant-scan pseudo (not found)",
		`{"pseudoIntensityReq":{"scanId": "non-existant-scan", "entries": {"indexes": [100,-1,104]}}}`,
		`{"msgId":1, "status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"pseudoIntensityResp":{}
		}`,
	)

	u1.AddSendReqAction("non-existant-scan spectra (1) (not found)",
		`{"spectrumReq":{"scanId": "non-existant-scan", "entries": {"indexes": [100,-1,104]}}}`,
		`{"msgId":2, "status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"spectrumResp":{}
		}`,
	)

	u1.AddSendReqAction("non-existant-scan spectra (2) (not found)",
		`{"spectrumReq":{"scanId": "non-existant-scan", "bulkSum": true, "maxValue": true}}`,
		`{"msgId":3, "status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"spectrumResp":{}
		}`,
	)

	u1.AddSendReqAction("non-existant-scan meta write (not found)",
		`{"scanMetaWriteReq":{"scanId": "non-existant-scan", "title": "Something", "description": "The blah"}}`,
		`{"msgId":4,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant-scan not found",
			"scanMetaWriteResp": {}
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
		`{"pseudoIntensityReq":{"scanId": "048300551", "entries": {"indexes": [100,-1,104]}}}`,
		`{
			"msgId": 1,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"pseudoIntensityResp": {}
		}`,
	)

	u1.AddSendReqAction("spectrum (expect no permission)",
		`{"spectrumReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{
			"msgId": 2,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"spectrumResp": {}
		}`,
	)

	u1.AddSendReqAction("spectrum bulk/max (expect no permission)",
		`{"spectrumReq":{"scanId": "048300551", "bulkSum": true, "maxValue": true}}`,
		`{
			"msgId": 3,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"spectrumResp": {}
		}`,
	)

	u1.AddSendReqAction("metaLabels (expect no permission)",
		`{"scanMetaLabelsAndTypesReq":{"scanId": "048300551"}}`,
		`{"msgId":4,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"scanMetaLabelsAndTypesResp":{}
		}`,
	)

	u1.AddSendReqAction("scanEntry (expect no permission)",
		`{"scanEntryReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":5,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"scanEntryResp":{}
		}`,
	)

	u1.AddSendReqAction("scanEntryMetadata (expect no permission)",
		`{"scanEntryMetadataReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":6,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"scanEntryMetadataResp":{}
		}`,
	)

	u1.AddSendReqAction("scanBeamLocations (expect no permission)",
		`{"scanBeamLocationsReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":7,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"scanBeamLocationsResp":{}
		}`,
	)

	u1.AddSendReqAction("imageListReq (expect no permission)",
		`{"imageListReq":{"scanIds": ["048300551"]}}`,
		`{"msgId":8,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"imageListResp":{}
		}`,
	)

	u1.AddSendReqAction("imageGetReq (expect no permission)",
		`{"imageGetReq":{"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
		`{"msgId":9,
			"status": "WS_NO_PERMISSION",
			"errorText": "User cannot access scan 048300551 associated with image 048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png. Error: View access denied for: OT_SCAN (048300551)",
			"imageGetResp":{}
		}`,
	)

	u1.AddSendReqAction("detectedDiffractionPeaksReq (expect no permission)",
		`{"detectedDiffractionPeaksReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":10,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: OT_SCAN (048300551)",
			"detectedDiffractionPeaksResp":{}
		}`,
	)

	u1.AddSendReqAction("scan meta write (not found)",
		`{"scanMetaWriteReq":{"scanId": "048300551", "title": "Something", "description": "The blah"}}`,
		`{"msgId":11,
			"status": "WS_NO_PERMISSION",
			"errorText": "Edit access denied for: OT_SCAN (048300551)",
			"scanMetaWriteResp": {}
		}`,
	)

	/*
		u1.AddSendReqAction("imageBeamLocationsReq (expect no permission)",
			`{"imageBeamLocationsReq":{"imageName": "PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
			`{"msgId":9,
				"status": "WS_NO_PERMISSION",
				"errorText": "View access denied for: OT_SCAN (048300551)",
				"imageBeamLocationsResp":{}
			}`,
		)
	*/
	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
}

func testScanDataHasPermission(apiHost string, actionMsg string, editAllowed bool) {
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
		`{"pseudoIntensityReq":{"scanId": "048300551", "entries": {"indexes": [100,-1,104]}}}`,
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
						"id": 189,
						"intensities${LIST,MODE=LENGTH,LENGTH=32}": []
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("Pseudo: "+actionMsg+" no indexes (should work)",
		`{"pseudoIntensityReq":{"scanId": "048300551"}}`,
		`{"msgId":2, "status": "WS_OK",
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
						"id": 93,
						"intensities${LIST,MODE=LENGTH,LENGTH=32}": []
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("Spectra: "+actionMsg+" (should work)",
		`{"spectrumReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":3, "status": "WS_OK",
			"spectrumResp":{
				"channelCount": 4096,
				"normalSpectraForScan": 242,
				"liveTimeMetaIndex": 119,
				"spectraPerLocation": [
					{
						"spectra": [
							{
								"detector": "A",
								"type": "SPECTRUM_NORMAL",
								"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
								"maxCount": 3096,
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
								"maxCount": 2855,
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
								"maxCount": 367901,
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
								"maxCount": 353738,
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
								"maxCount": 7017,
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
								"maxCount": 6786,
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

	u1.AddSendReqAction("Spectra: "+actionMsg+" (should work)",
		`{"spectrumReq":{"scanId": "048300551", "bulkSum": true, "maxValue": true, "entries": {"indexes": [] } }}`,
		`{"msgId":4, "status": "WS_OK",
			"spectrumResp":{
				"channelCount": 4096,
				"normalSpectraForScan": 242,
				"liveTimeMetaIndex": 119,
				"bulkSpectra": [
					{
						"detector": "A",
						"type": "SPECTRUM_BULK",
						"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
						"maxCount": 367901,
						"meta": "${IGNORE}"
					},
					{
						"detector": "B",
						"type": "SPECTRUM_BULK",
						"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
						"maxCount": 353738,
						"meta": "${IGNORE}"
					}
				],
				"maxSpectra": [
					{
						"detector": "A",
						"type": "SPECTRUM_MAX",
						"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
						"maxCount": 7017,
						"meta": {
							"119": {
								"fvalue": 14.293016
							},
							"120": {
								"fvalue": -11.8
							},
							"123": {
								"fvalue": 15
							},
							"124": {
								"fvalue": 7.9226
							},
							"125": {
								"svalue": "YY"
							},
							"126": {
								"svalue": "EMSA/MAS spectral data file"
							},
							"127": {
								"svalue": "2"
							},
							"128": {
								"svalue": "4096"
							},
							"129": {
								"svalue": "PIXL Flight Model"
							},
							"130": {
								"svalue": "XRF"
							},
							"131": {
								"svalue": "N/A"
							},
							"132": {
								"svalue": "TC202v2.0 PIXL"
							},
							"133": {
								"svalue": "eV"
							},
							"134": {
								"svalue": "-1.032"
							},
							"135": {
								"svalue": "COUNTS"
							}
						}
					},
					{
						"detector": "B",
						"type": "SPECTRUM_MAX",
						"counts${LIST,MODE=LENGTH,MINLENGTH=2000}": [],
						"maxCount": 6786,
						"meta": "${IGNORE}"
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("MetaLabels: "+actionMsg+" (should work)",
		`{"scanMetaLabelsAndTypesReq":{"scanId": "048300551"}}`,
		`{"msgId":5, "status": "WS_OK",
			"scanMetaLabelsAndTypesResp":{
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
				],
				"metaTypes": [
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_STRING",
				"MT_INT",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_INT",
				"MT_STRING",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_INT",
				"MT_STRING",
				"MT_FLOAT",
				"MT_FLOAT",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING",
				"MT_STRING"
				]
			}
		}`,
	)

	u1.AddSendReqAction("scanEntry (should work)",
		`{"scanEntryReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":6, "status": "WS_OK",
			"scanEntryResp":{
				"entries${LIST,MODE=CONTAINS,LENGTH=4}": [
					{
						"id": 216,
						"timestamp": 678034827,
						"images": 1,
						"normalSpectra": 2,
						"meta": true,
						"location": true,
						"pseudoIntensities": true
					},
					{
						"id": 217,
						"timestamp": 678034966,
						"meta": true
					},
					{
						"id": 218,
						"timestamp": 678035193,
						"images": 1,
						"bulkSpectra": 2,
						"maxSpectra": 2,
						"meta": true,
						"location": true
					},
					{
						"id": 219,
						"timestamp": 678035443,
						"meta": true
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("scanEntryMetadata (should work)",
		`{"scanEntryMetadataReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":7, "status": "WS_OK",
			"scanEntryMetadataResp":{
				"entries${LIST,MODE=CONTAINS,LENGTH=4}": [
					{
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

	u1.AddSendReqAction("scanBeamLocationsReq (should work)",
		`{"scanBeamLocationsReq":{"scanId": "048300551", "entries": {"indexes": [128,-1,131]}}}`,
		`{"msgId":8, "status": "WS_OK",
			"scanBeamLocationsResp":{
				"beamLocations": [
					{
						"x": -0.150853,
						"y": 0.131032,
						"z": 0.246621
					},
					{},
					{
						"x": -0.135756,
						"y": 0.131042,
						"z": 0.249732
					},
					{}
				]
			}
		}`,
	)

	u1.AddSendReqAction("detectedDiffractionPeaksReq (expect no permission)",
		`{"detectedDiffractionPeaksReq":{"scanId": "048300551", "entries": {"indexes": [100,-1,104]}}}`,
		`{"msgId": 9,
			"status": "WS_OK",
			"detectedDiffractionPeaksResp": {
				"peaksPerLocation": [
					{
						"id": "188",
						"peaks${LIST,MODE=CONTAINS,LENGTH=3}": [
							{
								"peakChannel": 204,
								"effectSize": 6.0991945,
								"baselineVariation": 0.19179639,
								"globalDifference": 0.06739717,
								"differenceSigma": 0.16368404,
								"peakHeight": 0.3579455,
								"detector": "A"
							}
						]
					},
					{
						"id": "189",
						"peaks${LIST,MODE=CONTAINS,LENGTH=5}": [
							{
								"peakChannel": 832,
								"effectSize": 10.04235,
								"baselineVariation": 0.24875456,
								"globalDifference": 0.06951845,
								"differenceSigma": 0.15115373,
								"peakHeight": 0.22210322,
								"detector": "A"
							}
						]
					},
					{
						"id": "190",
						"peaks${LIST,MODE=CONTAINS,LENGTH=3}": [
							{
								"peakChannel": 483,
								"effectSize": 6.8425426,
								"baselineVariation": 0.22377764,
								"globalDifference": 0.08196892,
								"differenceSigma": 0.1122962,
								"peakHeight": 0.2930711,
								"detector": "A"
							}
						]
					},
					{
						"id": "191",
						"peaks${LIST,MODE=CONTAINS,LENGTH=4}": [
							{
								"peakChannel": 833,
								"effectSize": 11.62446,
								"baselineVariation": 0.25955257,
								"globalDifference": 0.08716079,
								"differenceSigma": 0.12355343,
								"peakHeight": 0.38338855,
								"detector": "A"
							}
						]
					},
					{
						"id": "192",
						"peaks${LIST,MODE=CONTAINS,LENGTH=9}": [
							{
								"peakChannel": 230,
								"effectSize": 7.650696,
								"baselineVariation": 0.15942448,
								"globalDifference": 0.097010836,
								"differenceSigma": 0.099084,
								"peakHeight": 0.15445672,
								"detector": "A"
							}
						]
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("imageListReq (should work)",
		`{"imageListReq":{"scanIds": ["048300551"]}}`,
		`{"msgId":10,
			"status": "WS_OK",
			"imageListResp":{
				"images": [
					{
						"imagePath": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
						"source": "SI_INSTRUMENT",
						"width": 752,
						"height": 580,
						"fileSize": 240084,
						"purpose": "SIP_VIEWING",
						"associatedScanIds": [
							"048300551"
						],
						"originScanId": "048300551"
					},
					{
						"imagePath": "048300551/PCW_0125_0678032223_000RCM_N00417120483005510093075J02.png",
						"source": "SI_INSTRUMENT",
						"width": 752,
						"height": 580,
						"fileSize": 256736,
						"purpose": "SIP_VIEWING",
						"associatedScanIds": [
							"048300551"
						],
						"originScanId": "048300551"
					}
				]
			}
		}`,
	)

	u1.AddSendReqAction("imageGetReq (should work)",
		`{"imageGetReq":{"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
		`{"msgId":11,
			"status": "WS_OK",
			"imageGetResp":{
				"image": {
					"imagePath": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
					"source": "SI_INSTRUMENT",
					"width": 752,
					"height": 580,
					"fileSize": 240084,
					"purpose": "SIP_VIEWING",
					"associatedScanIds": [
						"048300551"
					],
					"originScanId": "048300551"
				}
			}
		}`,
	)

	u1.AddSendReqAction("imageBeamLocationsReq (bad image name)",
		`{"imageBeamLocationsReq":{"imageName": "non-existant.jpg"}}`,
		`{
			"msgId": 12,
			"status": "WS_NOT_FOUND",
			"errorText": "non-existant.jpg not found",
			"imageBeamLocationsResp": {}
		}`,
	)

	u1.AddSendReqAction("imageBeamLocationsReq (should work)",
		`{"imageBeamLocationsReq":{"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
		`{"msgId":13, "status": "WS_OK",
			"imageBeamLocationsResp":{
				"locations": {
					"imageName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png",
					"locationPerScan": [
						{
							"scanId": "048300551",
							"locations": [
								{},
								{},
								{},
								{
									"i": 361.19135,
									"j": 293.5299
								},
								{},
								{
									"i": 65.51854,
									"j": 305.2161
								}
							]
						}
					]
				}
			}
		}`,
	)

	if editAllowed {
		u1.AddSendReqAction("scan meta write (should work)",
			`{"scanMetaWriteReq":{"scanId": "048300551", "title": "Naltsos", "description": "The first scan on Mars"}}`,
			`{"msgId":14,
				"status": "WS_OK",
				"scanMetaWriteResp":{}
			}`,
		)
	} else {
		u1.AddSendReqAction("scan meta write (not found)",
			`{"scanMetaWriteReq":{"scanId": "048300551", "title": "Something", "description": "The blah"}}`,
			`{"msgId":14,
				"status": "WS_NO_PERMISSION",
				"errorText": "Edit access denied for: OT_SCAN (048300551)",
				"scanMetaWriteResp": {}
			}`,
		)
	}

	// Changing default image and querying
	u1.AddSendReqAction("get default image (empty)",
		`{"imageGetDefaultReq":{"scanIds": ["048300551", "another"]}}`,
		`{"msgId":15,
			"status": "WS_NOT_FOUND",
			"errorText": "another not found",
			"imageGetDefaultResp": {}
		}`,
	)

	u1.AddSendReqAction("set default image",
		`{"imageSetDefaultReq":{"scanId": "048300551", "defaultImageFileName": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"}}`,
		`{"msgId":16,
			"status": "WS_OK",
			"imageSetDefaultResp": {}
		}`,
	)
	u1.AddSendReqAction("get default image (should work)",
		`{"imageGetDefaultReq":{"scanIds": ["048300551"]}}`,
		`{"msgId":17,
			"status": "WS_OK",
			"imageGetDefaultResp": {
				"defaultImagesPerScanId": {
					"048300551": "048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02.png"
				}
			}
		}`,
	)

	expectedMsgs := []string{
		`{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				]
			}
		}}`,
	}

	if editAllowed {
		expectedMsgs = append(expectedMsgs, `{"notificationUpd": {
			"notification": {
				"notificationType": "NT_SYS_DATA_CHANGED",
				"scanIds": [
					"048300551"
				]
			}
		}}`)
	}

	u1.CloseActionGroup(expectedMsgs, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)

	// Test the GET HTTP endpoint, this has nothing to do with users/websockets above
	testImageGet_OK(apiHost, imageGetJWT)

	// Delete cached images from S3
	err := apiStorageFileAccess.DeleteObject(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width200.png")
	failIf(err != nil && !apiStorageFileAccess.IsNotFoundError(err), fmt.Errorf("Failed to delete previous cached image for GET thumbnail test: %v", err))
	err = apiStorageFileAccess.DeleteObject(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width400.png")
	failIf(err != nil && !apiStorageFileAccess.IsNotFoundError(err), fmt.Errorf("Failed to delete previous cached image for GET thumbnail test: %v", err))
	err = apiStorageFileAccess.DeleteObject(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width1000.png")
	failIf(err != nil && !apiStorageFileAccess.IsNotFoundError(err), fmt.Errorf("Failed to delete previous cached image for GET thumbnail test: %v", err))
	err = apiStorageFileAccess.DeleteObject(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width1200.png")
	failIf(err != nil && !apiStorageFileAccess.IsNotFoundError(err), fmt.Errorf("Failed to delete previous cached image for GET thumbnail test: %v", err))

	////////////////////////////////
	// Generate small thumbnail!

	// Do the GET call, which should generate the image
	testImageGetScaled_OK(apiHost, imageGetJWT, 12, 200, 154)

	// Check that the file was created
	exists, err := apiStorageFileAccess.ObjectExists(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width200.png")
	failIf(err != nil, fmt.Errorf("Failed to check generated GET thumbnail exists: %v", err))
	failIf(!exists, fmt.Errorf("generated GET thumbnail not found in Image-Cache/ width 200"))

	// Now run it again, because this time it should be using the cached copy
	testImageGetScaled_OK(apiHost, imageGetJWT, 12, 200, 154)

	////////////////////////////////
	// Now try larger image!

	// Do the GET call, which should generate the image
	testImageGetScaled_OK(apiHost, imageGetJWT, 450, 400, 308)

	// Check that the file was created
	exists, err = apiStorageFileAccess.ObjectExists(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width400.png")
	failIf(err != nil, fmt.Errorf("Failed to check generated GET thumbnail exists: %v", err))
	failIf(!exists, fmt.Errorf("generated GET thumbnail not found in Image-Cache/ width 400"))

	// Now run it again, because this time it should be using the cached copy
	testImageGetScaled_OK(apiHost, imageGetJWT, 450, 400, 308)

	////////////////////////////////
	// Check that we don't scale up!
	testImageGetScaled_OK(apiHost, imageGetJWT, 1050, 752, 580)

	// Check that the file was NOT created
	exists, err = apiStorageFileAccess.ObjectExists(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width1000.png")
	failIf(err == nil && exists, fmt.Errorf("unexpected 1000 sized GET thumbnail not found in Image-Cache/. Error was: %v", err))

	exists, err = apiStorageFileAccess.ObjectExists(apiDatasetBucket, "Image-Cache/048300551/PCW_0125_0678031992_000RCM_N00417120483005510091075J02-width1200.png")
	failIf(err == nil && exists, fmt.Errorf("unexpected 1200 sized GET thumbnail not found in Image-Cache/. Error was: %v", err))
}
