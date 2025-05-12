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
