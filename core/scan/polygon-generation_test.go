package scan

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Example_scan_GeneratePolygons() {
	scanId := "189137412"
	imgName := "189137412/PCO_0545_0715357896_000RCM_N02651501891374120096075J__.png"
	beamVersion := 3

	db := wstestlib.GetDBWithEnvironment("local-pixlise")
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ScansName)
	result := coll.FindOne(ctx, bson.M{"_id": "189137412"})

	if result.Err() != nil {
		panic(result.Err())
	} else {
		scan := protos.ScanItem{}
		err := result.Decode(&scan)
		if err != nil {
			panic(err)
		}
	}

	fs := fileaccess.FSAccess{}
	fileBytes, err := fs.ReadObject("./test-data/dataset-"+scanId+".bin", "")
	if err != nil {
		panic(err)
	}

	// Now decode the data & return it
	datasetPB := &protos.Experiment{}
	err = proto.Unmarshal(fileBytes, datasetPB)
	if err != nil {
		panic(err)
	}

	fmt.Println("read ok")

	scanItemStr := `{
  "id": "189137412",
  "title": "Cal Target",
  "description": "",
  "dataTypes": [
    {
      "dataType": 1,
      "count": "55"
    },
    {
      "dataType": 2,
      "count": "1730"
    }
  ],
  "instrument": 1,
  "instrumentConfig": "PIXL",
  "timestampUnixSec": 1731028677,
  "meta": {
    "Site": "",
    "Sol": "0545",
    "RTT": "189137412",
    "SCLK": "715356580",
    "TargetId": "?",
    "SiteId": "26",
    "DriveId": "5150",
    "Target": ""
  },
  "contentCounts": {
    "NormalSpectra": 1730,
    "DwellSpectra": 0,
    "BulkSpectra": 2,
    "MaxSpectra": 2,
    "PseudoIntensities": 865
  },
  "creatorUserId": "PIXLISEImport",
  "tags": [
    "7qndwrc4z3ptkcpo"
  ],
  "completeTimeStampUnixSec": 1731016077
}`
	scanItem := &protos.ScanItem{}
	err = protojson.Unmarshal([]byte(scanItemStr), scanItem)
	if err != nil {
		panic(err)
	}

	indexes := []uint32{}
	// Use all indexes available in the file
	for c := range datasetPB.Locations {
		indexes = append(indexes, uint32(c))
	}

	scanEntries, err := ReadScanEntries(datasetPB, indexes)
	if err != nil {
		panic(err)
	}

	xyz := ReadXYZ(datasetPB, indexes)
	ij := []*protos.Coordinate2D{}

	coll = db.Collection(dbCollections.ImageBeamLocationsName)

	// Read the image and check that the user has access to all scans associated with it
	imgResult := coll.FindOne(ctx, bson.M{"_id": imgName})

	var dbLocs *protos.ImageLocations
	err = imgResult.Decode(&dbLocs)
	if err != nil {
		panic(err)
	}

	for _, loc := range dbLocs.LocationPerScan {
		if loc.ScanId == scanId && loc.BeamVersion == uint32(beamVersion) {
			ij = loc.Locations
			break
		}
	}

	if len(ij) <= 0 {
		panic("No ijs loaded")
	}

	configStr := `{
    "id": "PIXL",
    "minElement": 11,
    "maxElement": 92,
    "xrfeVLowerBound": 800,
    "xrfeVUpperBound": 20000,
    "xrfeVResolution": 230,
    "windowElement": 14,
    "tubeElement": 45,
    "defaultParams": "",
    "mmBeamRadius": 0.05999999865889549,
    "elevAngle": 70
}`
	config := &protos.DetectorConfig{}
	err = protojson.Unmarshal([]byte(configStr), config)
	if err != nil {
		panic(err)
	}

	err = GeneratePolygons(
		imgName,
		scanItem,
		scanEntries,
		xyz,
		beamVersion,
		&ij,
		config,
	)

	fmt.Printf("%v\n", err)

	// Output:
	// done
}
