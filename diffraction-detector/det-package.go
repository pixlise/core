// This is a go package, file name can be anything, any number of files within this directory, all need to say package <dirname>
// as first line. Names with capital letters are exported (so "public" in some ways, usable from other packages)
// We also now include the following license:

// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package detection

import (
	"errors"
	"io/ioutil"
	"math"

	protos "github.com/pixlise/core/generated-protos"

	"google.golang.org/protobuf/proto"
)

func BuildDiffractionProtobuf(dataset *protos.Experiment, diffractionData map[string][]DiffractionPeak) *protos.Diffraction {
	diffractionPB := &protos.Diffraction{}

	diffractionPB.TargetId = dataset.TargetId
	diffractionPB.DriveId = dataset.DriveId
	diffractionPB.SiteId = dataset.SiteId
	diffractionPB.Target = dataset.Target
	diffractionPB.Site = dataset.Site
	diffractionPB.Title = dataset.Title
	diffractionPB.Sol = dataset.Sol
	diffractionPB.Rtt = dataset.Rtt
	diffractionPB.Sclk = dataset.Sclk

	for loc := range diffractionData {
		peakData := make([]*protos.Diffraction_Location_Peak, len(diffractionData[loc]))
		for peakID, peak := range diffractionData[loc] {
			peakPB := protos.Diffraction_Location_Peak{}
			peakPB.PeakChannel = int32(peak.PeakChannel)
			peakPB.EffectSize = float32(peak.EffectSize)
			peakPB.BaselineVariation = float32(peak.BaselineVariation)
			peakPB.GlobalDifference = float32(peak.GlobalDifference)
			peakPB.DifferenceSigma = float32(peak.DifferenceSigma)
			peakPB.PeakHeight = float32(peak.PeakHeight)

			peakData[peakID] = &peakPB
		}
		locationData := protos.Diffraction_Location{}
		locationData.Id = loc
		locationData.Peaks = peakData
		diffractionPB.Locations = append(diffractionPB.Locations, &locationData)
	}

	return diffractionPB
}

func SaveDiffractionProtobuf(diffractionPB *protos.Diffraction, fname string) error {
	out, err := proto.Marshal(diffractionPB)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(fname, out, 0644); err != nil {
		return err
	}

	return nil
}

func ParseDiffractionProtoBuf(path string) (*protos.Diffraction, error) {
	diffractionData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	diffractionPB := &protos.Diffraction{}
	err = proto.Unmarshal(diffractionData, diffractionPB)
	if err != nil {
		return nil, err
	}
	return diffractionPB, nil
}

func ScanDataset(dataset *protos.Experiment) (map[string][]DiffractionPeak, error) {
	datasetDiffractionPeaksMap := make(map[string][]DiffractionPeak)
	for loc := range dataset.Locations {
		if len(dataset.Locations[loc].Detectors) == 2 { // this check is to avoid dwell/bulksum spectra
			aSpectrum := DecodeZeroRun(dataset.Locations[loc].Detectors[0].Spectrum)
			bSpectrum := DecodeZeroRun(dataset.Locations[loc].Detectors[1].Spectrum)

			peaks, err := ScanSpectra(aSpectrum, bSpectrum)
			if err == nil {
				if len(peaks) > 0 {
					locationID := dataset.Locations[loc].Id
					datasetDiffractionPeaksMap[locationID] = peaks
				}
			} else {
				return nil, err
			}
		}
	}
	return datasetDiffractionPeaksMap, nil
}

type DiffractionPeak struct {
	PeakChannel       int
	EffectSize        float64
	BaselineVariation float64
	GlobalDifference  float64
	DifferenceSigma   float64
	PeakHeight        float64
}

func DecodeZeroRun(encodedSpectrum []int32) []int32 {
	fullSpectrum := make([]int32, 4096)
	currentIndex := 0
	for i := 0; i < len(encodedSpectrum); i++ {
		if encodedSpectrum[i] != 0 {
			fullSpectrum[currentIndex] = encodedSpectrum[i]
			currentIndex++
		} else {
			currentIndex += int(encodedSpectrum[i+1])
			i++
		}
	}
	return fullSpectrum
}

func ScanSpectra(a []int32, b []int32) ([]DiffractionPeak, error) {
	if (len(a) != 4096) || (len(b) != 4096) {
		return nil, errors.New("a and b spectra lengths are not both of the expected size (4096)")
	}

	const halfResolution = 15
	const minAvgCount = 2.0
	const minEffect = 6.0
	const minChannel = 100
	const maxChannel = 2000
	const minHeight = 0.1

	potentialPeaks := []DiffractionPeak{}

	logA := make([]float64, 4096)
	logB := make([]float64, 4096)

	for i := range a {
		logA[i] = math.Log1p(float64(a[i]))
		logB[i] = math.Log1p(float64(b[i]))
	}

	normalizationFactor := 0.0
	allDiffs := make([]float64, 4096)
	for i := range a {
		allDiffs[i] = logA[i] - logB[i]
	}
	normalizationFactor = meanFloats(allDiffs)

	for i := minChannel; i < maxChannel; i++ {

		differences := make([]float64, 2*halfResolution)
		for j := i; j < i+2*halfResolution; j++ {
			differences[j-i] = (logA[j] - logB[j] - normalizationFactor)
		}

		meanDiff := meanFloats(differences)
		stdDiff := stdFloats(differences, meanDiff)
		tStatistic := math.Abs(meanDiff / (stdDiff / math.Sqrt(float64(2*halfResolution))))
		peakHeight := math.Abs(differences[halfResolution-1]) + math.Abs(differences[halfResolution]) - math.Abs(differences[0]) - math.Abs(differences[2*halfResolution-1])

		meanA := meanInts(a[i : i+2*halfResolution])
		meanB := meanInts(b[i : i+2*halfResolution])

		avgCounts := 0.5 * (meanA + meanB)

		baselineVariation := 0.0
		if meanA >= meanB {
			meanLogB := meanFloats(logB[i : i+2*halfResolution])
			baselineVariation = stdFloats(logB[i:i+2*halfResolution], meanLogB) / meanLogB
		} else {
			meanLogA := meanFloats(logA[i : i+2*halfResolution])
			baselineVariation = stdFloats(logA[i:i+2*halfResolution], meanLogA) / meanLogA
		}

		if avgCounts >= minAvgCount && tStatistic >= minEffect && peakHeight >= minHeight {
			potentialPeaks = append(potentialPeaks, DiffractionPeak{i + halfResolution, tStatistic, baselineVariation, math.Abs(normalizationFactor), stdDiff, peakHeight})
		}

	}
	peaks := pruneNeighbors(potentialPeaks, halfResolution)

	return peaks, nil
}

func meanInts(s []int32) float64 {
	if len(s) == 0 {
		return math.NaN()
	}
	var sum int32 = 0
	for _, v := range s {
		sum += v
	}
	return float64(sum) / float64(len(s))
}

func meanFloats(s []float64) float64 {
	if len(s) == 0 {
		return math.NaN()
	}
	var sum float64 = 0
	for _, v := range s {
		sum += v
	}
	return sum / float64(len(s))
}

func stdFloats(s []float64, mean float64) float64 {
	var std float64 = 0
	for _, v := range s {
		std += math.Pow(v-mean, 2)
	}
	return math.Sqrt(std / float64(len(s)-1))
}

func pruneNeighbors(potentialPeaks []DiffractionPeak, boundary int) []DiffractionPeak {
	peaks := []DiffractionPeak{}
	skip := false
	for i := 0; i < len(potentialPeaks); i++ {
		localMax := true
		for j := 0; j < len(potentialPeaks); j++ {
			dist := potentialPeaks[i].PeakChannel - potentialPeaks[j].PeakChannel
			if dist < 0 {
				dist = -dist
			}
			if i != j && dist <= boundary && (potentialPeaks[i].EffectSize < potentialPeaks[j].EffectSize) {
				localMax = false
			} else if i > j && dist <= boundary && (potentialPeaks[i].EffectSize == potentialPeaks[j].EffectSize) {
				localMax = true
			}
		}
		if localMax {
			peaks = append(peaks, potentialPeaks[i])
		} else {
			skip = true
		}
	}
	if skip {
		return pruneNeighbors(peaks, boundary)
	}

	return peaks
}
