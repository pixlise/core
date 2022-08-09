package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	datasetModel "gitlab.com/pixlise/pixlise-go-api/core/dataset"
	diffractionDetection "gitlab.com/pixlise/pixlise-go-api/diffraction-detector"
)

func main() {
	var argPath = flag.String("path", "", "Path to file")
	var locationID = flag.String("query", "", "Query location to show primary peak")
	var savePath = flag.String("save", "", "Path to save protobuf binary")
	var loadPath = flag.String("load", "", "Path to load protobuf binary")

	flag.Parse()

	if nil != loadPath && *loadPath != "" {
		peaksParsed, err := diffractionDetection.ParseDiffractionProtoBuf(*loadPath)
		if err == nil {
			fmt.Println(peaksParsed.Title)
			fmt.Printf("%v Locations with Peaks found\n", len(peaksParsed.Locations))
			for _, loc := range peaksParsed.Locations {
				fmt.Printf("Location: %v", loc.Id)
				fmt.Print("\tPeaks: [")
				for _, p := range loc.Peaks {
					fmt.Printf("{%v  %v %v}  ", p.PeakChannel, p.EffectSize, p.BaselineVariation)
				}
				fmt.Println("]")
			}
		} else {
			fmt.Printf("Error loading diffraction file!\n %v\n", err)
		}

	} else {
		protoParsed, err := datasetModel.ReadDatasetFile(*argPath)

		if err != nil {
			fmt.Printf("Failed to open file \"%v\": \"%v\"\n", *argPath, err)
			os.Exit(1)
		}
		fmt.Printf("Opened %v, got RTT: %v\n", *argPath, protoParsed.Rtt)
		fmt.Println(protoParsed.Title)

		fmt.Println("Scanning dataset for diffraction peaks")
		datasetPeaks, err := diffractionDetection.ScanDataset(protoParsed)
		if err == nil {
			fmt.Println("Completed scan successfully")
		} else {
			fmt.Println("Error Encoundered During Scanning!")
			fmt.Println(err)
			os.Exit(1)
		}

		if nil != locationID && *locationID != "" {
			if *locationID == "ALL" {

				locs := make([]string, 0, len(datasetPeaks))
				for loc := range datasetPeaks {
					locs = append(locs, loc)
				}
				fmt.Printf("%v/%v Locations with Diffraction Peaks!\n", len(locs), len(protoParsed.Locations))
				sort.Slice(locs, func(i, j int) bool {
					sort.Slice(datasetPeaks[locs[i]], func(k, l int) bool {
						return datasetPeaks[locs[i]][k].EffectSize > datasetPeaks[locs[i]][l].EffectSize
					})
					sort.Slice(datasetPeaks[locs[j]], func(k, l int) bool {
						return datasetPeaks[locs[j]][k].EffectSize > datasetPeaks[locs[j]][l].EffectSize
					})
					return datasetPeaks[locs[i]][0].EffectSize > datasetPeaks[locs[j]][0].EffectSize
				})
				for _, loc := range locs {
					fmt.Printf("Location: %v\tPeaks:%v\n", loc, datasetPeaks[loc])
				}
			} else {
				peaks := datasetPeaks[*locationID]
				sort.Slice(peaks, func(i, j int) bool {
					return peaks[i].EffectSize > peaks[j].EffectSize
				})
				fmt.Println(*locationID)
				fmt.Println(peaks)
			}
		}

		if nil != savePath && *savePath != "" {
			fmt.Println("Saving binary file")
			diffractionPB := diffractionDetection.BuildDiffractionProtobuf(protoParsed, datasetPeaks)
			err := diffractionDetection.SaveDiffractionProtobuf(diffractionPB, *savePath)
			if err == nil {
				fmt.Println("File saved successfully")
			} else {
				fmt.Println("Error Encoundered During Saving!")
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}
