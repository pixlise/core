package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func main() {
	// Try to load the config file
	apiClient, err := client.Authenticate()
	fmt.Printf("auth err: %v\n", err)
	if err != nil {
		return
	}

	err = apiClient.UploadImage(&protos.ImageUploadHttpRequest{
		Name:         "myimage.png",
		ImageData:    []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa, 0x0, 0x0, 0x0, 0xd, 0x49, 0x48, 0x44, 0x52, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1, 0x8, 0x2, 0x0, 0x0, 0x0, 0x90, 0x77, 0x53, 0xde, 0x0, 0x0, 0x0, 0x1, 0x73, 0x52, 0x47, 0x42, 0x0, 0xae, 0xce, 0x1c, 0xe9, 0x0, 0x0, 0x0, 0xc, 0x49, 0x44, 0x41, 0x54, 0x18, 0x57, 0x63, 0x28, 0x3d, 0xaf, 0xb, 0x0, 0x3, 0x2e, 0x1, 0x72, 0x50, 0x4e, 0xda, 0xdf, 0x0, 0x0, 0x0, 0x0, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82},
		OriginScanId: "069927431",
	})

	return

	err = apiClient.SaveMapData("my-data", &protos.ClientMap{
		EntryPMCs:   []int32{7, 8, 10, 12},
		FloatValues: []float64{79.4, 89.4, 109.4, 129.4},
	})
	fmt.Printf("SaveMapData: %v\n", err)

	mapData, err := apiClient.LoadMapData("my-data")
	fmt.Printf("LoadMapData: %v|%v\n", err, mapData)

	return

	// Dev: 500302337 is missing bulk sum?

	spectrum, err := apiClient.GetScanSpectrum("261161477", 15, protos.SpectrumType_SPECTRUM_NORMAL, "A")
	fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

	spectrum, err = apiClient.GetScanSpectrum("261161477", 8383824, protos.SpectrumType_SPECTRUM_BULK, "B")
	fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

	rangeMap, err := apiClient.GetScanSpectrumRangeAsMap("475070977", 300, 302, "B")
	fmt.Printf("%v|%v\n", err, len(rangeMap.EntryPMCs))

	// diffMap, err := apiClient.GetDiffractionAsMap("475070977", protos.EnergyCalibrationSource_CAL_BULK_SUM, 0, 4096)
	// fmt.Printf("err: %v\n", err)
	// if diffMap != nil {
	// 	fmt.Printf("PMC, Value\n")
	// 	for k, v := range diffMap.EntryPMCs {
	// 		fmt.Printf("All Points,%v,%v\n", v, diffMap.FloatValues[k])
	// 	}
	// }

	// ruffMap, err := apiClient.GetRoughnessAsMap("475070977", protos.EnergyCalibrationSource_CAL_BULK_SUM)
	// fmt.Printf("err: %v\n", err)
	// if ruffMap != nil {
	// 	fmt.Printf("PMC, Value\n")
	// 	for k, v := range ruffMap.EntryPMCs {
	// 		fmt.Printf("All Points,%v,%v\n", v, ruffMap.FloatValues[k])
	// 	}
	// }
	return

	xyzs, err := apiClient.GetScanBeamLocations("261161477")
	fmt.Print(len(xyzs.Locations))

	diff, err := apiClient.GetDiffractionPeaks("261161477", protos.EnergyCalibrationSource_CAL_BULK_SUM)
	fmt.Printf("err: %v\n", err)
	if diff != nil {
		fmt.Printf("num peaks: %v\n", len(diff.Peaks))
		fmt.Printf("num roughness: %v\n", len(diff.Roughnesses))
		fmt.Printf("first data: %v\n", diff.Peaks[0])
	}
	//fmt.Printf("data: %v\n", diff)

	_, err = apiClient.SetUserScanCalibration("261161477", "A", -18.500, 7.862)
	fmt.Printf("err: %v\n", err)

	diff, err = apiClient.GetDiffractionPeaks("261161477", protos.EnergyCalibrationSource_CAL_USER)
	fmt.Printf("err: %v\n", err)
	if diff != nil {
		fmt.Printf("num peaks: %v\n", len(diff.Peaks))
		fmt.Printf("num roughness: %v\n", len(diff.Roughnesses))
		fmt.Printf("first data: %v\n", diff.Peaks[0])
	}
	//fmt.Printf("data: %v\n", diff)
}
