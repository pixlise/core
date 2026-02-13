package scan

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func Example_scan_GeneratePolygons() {
	scanId := "189137412"
	imgName := "189137412/PCO_0545_0715357896_000RCM_N02651501891374120096075J__.png"
	beamVersion := 3

	// db := wstestlib.GetDBWithEnvironment("local-pixlise")
	// ctx := context.TODO()
	// coll := db.Collection(dbCollections.ScansName)
	/*	result := coll.FindOne(ctx, bson.M{"_id": "189137412"})

		if result.Err() != nil {
			panic(result.Err())
		} else {
			scan := protos.ScanItem{}
			err := result.Decode(&scan)
			if err != nil {
				panic(err)
			}
		}
	*/
	fs := fileaccess.FSAccess{}
	fileBytes, err := fs.ReadObject("./test-data/"+scanId+"-dataset.bin", "")
	if err != nil {
		panic(err)
	}

	// Now decode the data & return it
	datasetPB := &protos.Experiment{}
	err = proto.Unmarshal(fileBytes, datasetPB)
	if err != nil {
		panic(err)
	}

	scanItemData, err := os.ReadFile("./test-data/189137412-scan.json")
	if err != nil {
		panic(err)
	}

	scanItem := &protos.ScanItem{}
	err = protojson.Unmarshal(scanItemData, scanItem)
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

	ijData, err := os.ReadFile("./test-data/189137412-ijs.json")
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(ijData, &ij)
	if err != nil {
		panic(err)
	}
	/*
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
	*/

	// d, _ := json.Marshal(ij)
	// os.WriteFile("./test-data/189137412-ijs.json", d, 777)

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

	// The output text is comparing the chrome debug console output with what happened in Go. Changes
	// allowed for:
	// - Go outputs: Location data physical size X=0.024586006999015808, Y=0.016534999012947083, Z=5.50001859664917e-05
	//   where Z is the same as Z=0.0000550001859664917 so changing the expected value to the Go output
	// - The original code randomly sampled 1000 PMCs to find the min distance and work out a display point radius. We
	//   don't random sample, we're server side so have time, and we check each point. This might get slow for large
	//   scans but we'll deal with that when it happens. For now, the min distance of this scan is 2.543132907103864

	// Output:
	//   Location position relative to context image: (x,y)=49,135, (w,h)=530,339
	//   Location data physical size X=0.024586006999015808, Y=0.016534999012947083, Z=5.50001859664917e-05
	//   Beam location is in meters
	//   Conversion factor for image pixels to mm: 0.04709422455222193
	//   Generated locationDisplayPointRadius: 2.543132907103864
	//   Point cluster 1 contains 25 PMCs, 12 footprint points, 0.000 degrees rotated
	//   Point cluster 2 contains 51 PMCs, 12 footprint points, -39.878 degrees rotated
	//   Point cluster 3 contains 51 PMCs, 14 footprint points, 49.857 degrees rotated
	//   Point cluster 4 contains 51 PMCs, 14 footprint points, -40.000 degrees rotated
	//   Point cluster 5 contains 25 PMCs, 12 footprint points, 0.000 degrees rotated
	//   Point cluster 6 contains 25 PMCs, 14 footprint points, 0.000 degrees rotated
	//   Point cluster 7 contains 25 PMCs, 11 footprint points, 0.000 degrees rotated
	//   Point cluster 8 contains 51 PMCs, 14 footprint points, 39.928 degrees rotated
	//   Point cluster 9 contains 51 PMCs, 15 footprint points, 40.075 degrees rotated
	//   Point cluster 10 contains 51 PMCs, 13 footprint points, 40.234 degrees rotated
	//   Point cluster 11 contains 153 PMCs, 16 footprint points, 126.771 degrees rotated
	//   Point cluster 12 contains 153 PMCs, 15 footprint points, 39.604 degrees rotated
	//   Point cluster 13 contains 51 PMCs, 15 footprint points, 42.187 degrees rotated
	//   Point cluster 14 contains 51 PMCs, 14 footprint points, 42.403 degrees rotated
	//   Point cluster 15 contains 51 PMCs, 14 footprint points, 42.584 degrees rotated
	// <nil>
}

/* Text from PIXLISE in chrome dev console:

 Location position relative to context image: (x,y)=49,135, (w,h)=530,339
context-image-scan-model-generator.ts:379   Location data physical size X=0.024586006999015808, Y=0.016534999012947083, Z=0.0000550001859664917
context-image-scan-model-generator.ts:537   Beam location is in meters
context-image-scan-model-generator.ts:120   Conversion factor for image pixels to mm: 0.04709422455222193
context-image-scan-model-generator.ts:495   Generated locationDisplayPointRadius: 2.668047466338241
context-image-scan-model-generator.ts:680   Point cluster 1 contains 25 PMCs, 12 footprint points, 0.000 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 2 contains 51 PMCs, 12 footprint points, -39.878 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 3 contains 51 PMCs, 14 footprint points, 49.857 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 4 contains 51 PMCs, 14 footprint points, -40.000 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 5 contains 25 PMCs, 12 footprint points, 0.000 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 6 contains 25 PMCs, 14 footprint points, 0.000 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 7 contains 25 PMCs, 11 footprint points, 0.000 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 8 contains 51 PMCs, 14 footprint points, 39.928 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 9 contains 51 PMCs, 15 footprint points, 40.075 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 10 contains 51 PMCs, 13 footprint points, 40.234 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 11 contains 153 PMCs, 16 footprint points, 126.771 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 12 contains 153 PMCs, 15 footprint points, 39.604 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 13 contains 51 PMCs, 15 footprint points, 42.187 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 14 contains 51 PMCs, 14 footprint points, 42.403 degrees rotated
context-image-scan-model-generator.ts:680   Point cluster 15 contains 51 PMCs, 14 footprint points, 42.584 degrees rotated

*/
