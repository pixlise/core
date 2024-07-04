package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Example_updateScanImageDataTypes() {
	db := wstestlib.GetDB()
	ctx := context.TODO()

	// Prep data
	db.Collection(dbCollections.ScansName).Drop(ctx)
	db.Collection(dbCollections.ImagesName).Drop(ctx)

	l := &logger.StdOutLoggerForTest{}
	fmt.Printf("Non-existant: %v\n", UpdateScanImageDataTypes("non-existant", db, l))

	// Add scan
	scanItem := &protos.ScanItem{
		Id:          "052822532",
		Title:       "Beaujeu",
		Description: "",
		DataTypes: []*protos.ScanItem_ScanTypeCount{
			{
				DataType: protos.ScanDataType_SD_XRF,
				Count:    234,
			},
		},
		Instrument:       1,
		InstrumentConfig: "PIXL",
		TimestampUnixSec: 1707463677,
		Meta: map[string]string{
			"Site":     "",
			"RTT":      "052822532",
			"SCLK":     "679215716",
			"Sol":      "0138",
			"DriveId":  "1812",
			"TargetId": "?",
			"Target":   "",
			"SiteId":   "5",
		},
		ContentCounts: map[string]int32{
			"MaxSpectra":        2,
			"PseudoIntensities": 225,
			"NormalSpectra":     450,
			"DwellSpectra":      0,
			"BulkSpectra":       2,
		},
	}

	scanInsert, err := db.Collection(dbCollections.ScansName).InsertOne(ctx, scanItem, options.InsertOne())
	fmt.Printf("Scan insert err: %v, scanInsert: %+v\n", err, scanInsert)

	fmt.Printf("Zero values: %v\n", UpdateScanImageDataTypes(scanItem.Id, db, l))
	scanRead, err := scan.ReadScanItem(scanItem.Id, db)
	fmt.Printf("SavedScanCounts: %v Err: %v\n", scanTypeToString(scanRead.DataTypes), err)

	// Insert image
	img := &protos.ScanImage{
		ImagePath:         "052822532/PCW_0138_0679216324_000RCM_N00518120528225320077075J03.png",
		Source:            1,
		Width:             752,
		Height:            580,
		FileSize:          230396,
		Purpose:           protos.ScanImagePurpose_SIP_VIEWING,
		AssociatedScanIds: []string{"052822532"},
		OriginScanId:      "052822532",
	}

	imgInsert, err := db.Collection(dbCollections.ImagesName).InsertOne(ctx, img, options.InsertOne())
	fmt.Printf("Image insert err: %v, imgInsert: %+v\n", err, imgInsert)

	fmt.Printf("One image: %v\n", UpdateScanImageDataTypes(scanItem.Id, db, l))
	scanRead, err = scan.ReadScanItem(scanItem.Id, db)
	fmt.Printf("SavedScanCounts: %v Err: %v\n", scanTypeToString(scanRead.DataTypes), err)

	// Insert RGBU images
	rgbu := &protos.ScanImage{
		ImagePath:         "052822532/PCCR0138_0679289188_000VIS_N005000005282253200770LUD01.tif",
		Source:            2,
		Width:             752,
		Height:            580,
		FileSize:          18173394,
		Purpose:           protos.ScanImagePurpose_SIP_MULTICHANNEL,
		AssociatedScanIds: []string{"052822532"},
		OriginScanId:      "052822532",
	}

	rgbuInsert, err := db.Collection(dbCollections.ImagesName).InsertOne(ctx, rgbu, options.InsertOne())
	fmt.Printf("RGBU Image insert err: %v, rgbuInsert: %+v\n", err, rgbuInsert)

	fmt.Printf("Two image: %v\n", UpdateScanImageDataTypes(scanItem.Id, db, l))
	scanRead, err = scan.ReadScanItem(scanItem.Id, db)
	fmt.Printf("SavedScanCounts: %v Err: %v\n", scanTypeToString(scanRead.DataTypes), err)

	// Output:
	// Non-existant: mongo: no documents in result
	// Scan insert err: <nil>, scanInsert: &{InsertedID:052822532}
	// Zero values: <nil>
	// SavedScanCounts:   SD_XRF=234 Err: <nil>
	// Image insert err: <nil>, imgInsert: &{InsertedID:052822532/PCW_0138_0679216324_000RCM_N00518120528225320077075J03.png}
	// One image: <nil>
	// SavedScanCounts:   SD_XRF=234   SD_XRF=234 SD_IMAGE=1 Err: <nil>
	// RGBU Image insert err: <nil>, rgbuInsert: &{InsertedID:052822532/PCCR0138_0679289188_000VIS_N005000005282253200770LUD01.tif}
	// Two image: <nil>
	// SavedScanCounts:   SD_XRF=234   SD_XRF=234 SD_IMAGE=1   SD_XRF=234   SD_XRF=234 SD_IMAGE=1 SD_RGBU=1 Err: <nil>
}

func scanTypeToString(dt []*protos.ScanItem_ScanTypeCount) string {
	result := ""
	for _, i := range dt {
		result += fmt.Sprintf(" %v %v=%v", result, i.DataType, i.Count)
	}
	return result
}
