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

	// spectrum, err := apiClient.GetScanSpectrum("261161477", 15, protos.SpectrumType_SPECTRUM_NORMAL, "A")
	// fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

	// spectrum, err = apiClient.GetScanSpectrum("261161477", 8383824, protos.SpectrumType_SPECTRUM_BULK, "B")
	// fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

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
