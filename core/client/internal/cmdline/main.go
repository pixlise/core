package main

import (
	"fmt"
	"log"
	"time"

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

	// roi, err := apiClient.GetROI("k0adnljtrszqv1e9", false)
	// fmt.Printf("getRGetROIOI: %v|%v", err, roi.RegionOfInterest.ScanEntryIndexesEncoded)

	// if len(roi.RegionOfInterest.ScanEntryIndexesEncoded) > 29 {
	// 	roi.RegionOfInterest.ScanEntryIndexesEncoded = []int32{659, 660, 661, 662, 663, 664, 665, 666,
	// 		667, 668, 669, 670, 671, 672, 673,
	// 		726, 727, 728, 729, 730, 731, 732,
	// 		733, 734, 735, 736, 737, 738, 739}
	// } else {
	// 	roi.RegionOfInterest.ScanEntryIndexesEncoded = []int32{406, 461, 462, 471, 472, 527, 528, 537,
	// 		538, 587, 588, 593, 594, 596, 601,
	// 		602, 603, 608, 609, 610, 611, 650,
	// 		651, 653, 654, 655, 656, 678, 679,
	// 		680, 681, 682, 683, 684, 685, 686,
	// 		714, 715, 716, 717, 718, 719, 720,
	// 		743, 744, 745, 746, 783, 784, 785,
	// 		786, 787, 788, 789, 790, 811}
	// }

	// roi.RegionOfInterest.Owner = nil

	// roiWriteResult, err := apiClient.CreateROI(roi.RegionOfInterest, false)
	// fmt.Printf("CreateROI: %v|%v", err, roiWriteResult)

	// return

	// err = apiClient.UploadImage(&protos.ImageUploadHttpRequest{
	// 	Name:         "myimage.png",
	// 	ImageData:    []byte{0x89, 0x50, 0x4e, 0x47, 0xd, 0xa, 0x1a, 0xa, 0x0, 0x0, 0x0, 0xd, 0x49, 0x48, 0x44, 0x52, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1, 0x8, 0x2, 0x0, 0x0, 0x0, 0x90, 0x77, 0x53, 0xde, 0x0, 0x0, 0x0, 0x1, 0x73, 0x52, 0x47, 0x42, 0x0, 0xae, 0xce, 0x1c, 0xe9, 0x0, 0x0, 0x0, 0xc, 0x49, 0x44, 0x41, 0x54, 0x18, 0x57, 0x63, 0x28, 0x3d, 0xaf, 0xb, 0x0, 0x3, 0x2e, 0x1, 0x72, 0x50, 0x4e, 0xda, 0xdf, 0x0, 0x0, 0x0, 0x0, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82},
	// 	OriginScanId: "069927431",
	// })

	// spectrum, err := apiClient.GetScanSpectrum("500302337", 4, protos.SpectrumType_SPECTRUM_NORMAL, "B")
	// fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum.Counts)
	// return

	// PIXLISE
	// Letters are pointing down...
	pixliseY := []float64{
		1, 2, 3, 4, 5, 6, 7, 8, 9,
		5, 9,
		5, 9,
		5, 9,
		6, 7, 8,

		1, 9,
		1, 2, 3, 4, 5, 6, 7, 8, 9,
		1, 9,

		1, 9,
		2, 8,
		3, 7,
		4, 6,
		5,
		4, 6,
		3, 7,
		2, 8,
		1, 9,

		1, 2, 3, 4, 5, 6, 7, 8, 9,
		1,
		1,
		1,

		1, 9,
		1, 2, 3, 4, 5, 6, 7, 8, 9,
		1, 9,

		2, 6, 7, 8,
		1, 5, 9,
		1, 5, 9,
		1, 5, 9,
		2, 3, 4, 8,

		1, 2, 3, 4, 5, 6, 7, 8, 9,
		1, 5, 9,
		1, 5, 9,
		1, 9,
	}

	pixliseX := []float64{
		1, 1, 1, 1, 1, 1, 1, 1, 1,
		2, 2,
		3, 3,
		4, 4,
		5, 5, 5,

		7, 7,
		8, 8, 8, 8, 8, 8, 8, 8, 8,
		9, 9,

		11, 11,
		12, 12,
		13, 13,
		14, 14,
		15,
		16, 16,
		17, 17,
		18, 18,
		19, 19,

		21, 21, 21, 21, 21, 21, 21, 21, 21,
		22,
		23,
		24,

		26, 26,
		27, 27, 27, 27, 27, 27, 27, 27, 27,
		28, 28,

		30, 30, 30, 30,
		31, 31, 31,
		32, 32, 32,
		33, 33, 33,
		34, 34, 34, 34,

		36, 36, 36, 36, 36, 36, 36, 36, 36,
		37, 37, 37,
		38, 38, 38,
		39, 39,
	}

	// for c := 0; c < len(pixliseX); c++ {
	// 	if pixliseX[c] > 3 {
	// 		pixliseX[c]++
	// 	}
	// }

	if len(pixliseX) != len(pixliseY) {
		log.Fatal("Lengths dont match")
	}

	// Add a top and bottom line to compress the letters in Y
	countPerLetter := []int32{}
	count := int32(0)
	for c, x := range pixliseX {
		if c > 0 && x-pixliseX[c-1] > 1 {
			countPerLetter = append(countPerLetter, count)
			count = 0
		}
		count++
	}

	columnCounts := []int32{}
	count = 0
	for c, y := range pixliseY {
		if c > 0 && y <= pixliseY[c-1] {
			columnCounts = append(columnCounts, count)
			count = 0
		}
		count++
	}
	// Add last column
	columnCounts = append(columnCounts, count)

	// Move everything up
	//pixliseX = append([]float64{0, 0}, pixliseX...)
	//pixliseY = append([]float64{0, 14}, pixliseY...)

	lastX := pixliseX[len(pixliseX)-1]
	dataX := []float64{0, float64(lastX)}
	dataY := []float64{0, 27}

	readPos := 0
	readCol := 0
	skipRead := 0
	for c := 0; c < int(lastX); c++ {
		if skipRead <= 0 && readCol < len(columnCounts) {
			for i := 0; i < int(columnCounts[readCol]); i++ {
				dataX = append(dataX, pixliseX[readPos]+float64(lastX)-float64(c))
				dataY = append(dataY, pixliseY[readPos]+9)
				readPos++
			}

			if readPos < len(pixliseX) {
				skipRead = int(pixliseX[readPos]-pixliseX[readPos-1]) - 1
			} else {
				skipRead = 0
			}

			readCol++
		} else {
			skipRead--
		}

		for i := 2; i < len(dataX); i++ {
			dataX[i]--
		}

		pmcs := []int32{}
		for c := range dataX {
			pmcs = append(pmcs, int32(c))
		}

		err = apiClient.SaveMapData("my-dataX", &protos.ClientMap{
			EntryPMCs:   pmcs,
			FloatValues: dataX,
		})

		err = apiClient.SaveMapData("my-dataY", &protos.ClientMap{
			EntryPMCs:   pmcs,
			FloatValues: dataY,
		})

		time.Sleep(time.Millisecond * 200)
	}
	/*
		err = apiClient.SaveMapData("my-data", &protos.ClientMap{
			EntryPMCs:   []int32{17000, 17002, 17003, 17005},
			FloatValues: []float64{79.4, 89.4, 120.4, 129.4},
		})
		fmt.Printf("saveMap my-data err: %v\n", err)

		err = apiClient.SaveMapData("my-data2", &protos.ClientMap{
			EntryPMCs:   []int32{17000, 17002, 17003, 17005},
			FloatValues: []float64{3.1415926, 5, 12.83, 23.21},
		})
		fmt.Printf("saveMap my-data2 err: %v\n", err)*/
	return

	// mapData, err := apiClient.LoadMapData("my-data")
	// fmt.Printf("LoadMapData: %v|%v\n", err, mapData)

	// return

	// // Dev: 500302337 is missing bulk sum?

	// spectrum, err := apiClient.GetScanSpectrum("261161477", 15, protos.SpectrumType_SPECTRUM_NORMAL, "A")
	// fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

	// spectrum, err = apiClient.GetScanSpectrum("261161477", 8383824, protos.SpectrumType_SPECTRUM_BULK, "B")
	// fmt.Printf("%v|%v|%v\n", err, len(spectrum.Counts), spectrum)

	// rangeMap, err := apiClient.GetScanSpectrumRangeAsMap("475070977", 300, 302, "B")
	// fmt.Printf("%v|%v\n", err, len(rangeMap.EntryPMCs))

	diffMap, err := apiClient.GetDiffractionAsMap("069927431", protos.EnergyCalibrationSource_CAL_BULK_SUM, 100, 120)
	fmt.Printf("err: %v\n", err)
	if diffMap != nil {
		fmt.Printf("PMC, Value\n")
		for k, v := range diffMap.EntryPMCs {
			fmt.Printf("All Points,%v,%v\n", v, diffMap.IntValues[k])
		}
	}

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
