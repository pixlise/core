package main

import (
	"context"
	"log"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

const scanWaitTime = 60 * 1000 * 10

func testScanData(apiHost string) {
	// Dataset - no permissions
	seedDBForScanData(nil, nil)
	userId := testScanDataBadId(apiHost, "Pseudo: ")
	testScanDataNoPermission(apiHost, "Pseudo: No permissions")

	// Dataset - user has userid viewer permissions
	seedDBForScanData(&protos.UserGroupList{
		UserIds:  []string{userId},
		GroupIds: []string{},
	}, nil)
	testScanDataHasPermission(apiHost, "Pseudo: UserId is scan viewer")

	// Dataset - no permissions (ensure above doesn't leak into the next test...)
	seedDBForScanData(nil, nil)
	testScanDataNoPermission(apiHost, "Pseudo: No permissions")

	// Dataset - user has groupid member permissions
	seedDBForScanData(&protos.UserGroupList{
		UserIds:  []string{},
		GroupIds: []string{"user-group-1"},
	}, nil)
	// Add the group id too!
	seedDBUserGroup(&protos.UserGroup{
		Id:             "user-group-1",
		Name:           "M2020 Scientists",
		CreatedUnixSec: 1234567890,
		Members: &protos.UserGroupList{
			UserIds: []string{userId},
		},
	})
	testScanDataHasPermission(apiHost, "Pseudo: UserId is member in UserGroup which is scan viewer")

	// Dataset - no permissions (ensure above doesn't leak into the next test...)
	seedDBForScanData(nil, nil)
	testScanDataNoPermission(apiHost, "Pseudo: No permissions")

	// Dataset - user has groupid viewer permissions
	seedDBForScanData(&protos.UserGroupList{
		UserIds:  []string{},
		GroupIds: []string{"user-group-1"},
	}, nil)
	// Add the group id too!
	seedDBUserGroup(&protos.UserGroup{
		Id:             "user-group-1",
		Name:           "M2020 Scientists",
		CreatedUnixSec: 1234567890,
		Viewers: &protos.UserGroupList{
			UserIds: []string{userId},
		},
	})
	testScanDataHasPermission(apiHost, "Pseudo: UserId is viewer in UserGroup which is scan viewer")
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

	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
	return u1.GetUserId()
}

func testScanDataNoPermission(apiHost string, actionMsg string) {
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
	u1.AddSendReqAction(actionMsg+" (expect no permission)",
		`{"pseudoIntensityReq":{"scanId": "048300551", "startingLocation": 100, "locationCount": 5}}`,
		`{
			"msgId": 1,
			"status": "WS_NO_PERMISSION",
			"errorText": "View access denied for: 048300551",
			"pseudoIntensityResp": {}
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
	u1.AddSendReqAction(actionMsg+" (should work)",
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

	u1.CloseActionGroup([]string{}, scanWaitTime)
	wstestlib.ExecQueuedActions(&u1)
}

func seedDBForScanData(viewers *protos.UserGroupList, editors *protos.UserGroupList) {
	// Insert a scan item into DB for this
	scanOwnerItem := protos.OwnershipItem{
		Id:             "scan_048300551",
		ObjectType:     protos.ObjectType_OT_SCAN,
		CreatorUserId:  "",
		CreatedUnixSec: 1646262426,
	}

	if viewers != nil {
		scanOwnerItem.Viewers = viewers
	}

	if editors != nil {
		scanOwnerItem.Editors = editors
	}

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

	coll = db.Collection(dbCollections.ScansName)
	err = coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, &scanItem)
	if err != nil {
		log.Fatalln(err)
	}
}

func seedDBUserGroup(group *protos.UserGroup) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.UserGroupsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	_, err = coll.InsertOne(ctx, group)
	if err != nil {
		log.Fatalln(err)
	}
}
