package client

import (
	"fmt"
	"strconv"

	protos "github.com/pixlise/core/v4/generated-protos"
)

var RoughnessItemThreshold = float32(0.16)
var DefaulteVCalibrationDetector = "A"

func ReadDiffractionData(
	detectedPeaks []*protos.DetectedDiffractionPerLocation,
	peakStatuses *protos.DetectedDiffractionPeakStatuses,
	spectrumEnergyCalibration *protos.ClientEnergyCalibration) ([]*protos.ClientDiffractionPeak, []*protos.ClientRoughnessItem) {
	// Some constants, along with others in this code!
	diffractionPeakHalfWidth := float32(15) * 0.5

	allPeaks := []*protos.ClientDiffractionPeak{}

	roughnessItems := []*protos.ClientRoughnessItem{}
	roughnessPMCs := map[int]bool{}

	for _, item := range detectedPeaks {
		pmc, err := strconv.Atoi(item.Id)
		if err != nil {
			fmt.Printf("Warning: Diffraction data contained invalid location id: %v", item.Id)
			continue
		}

		for _, peak := range item.Peaks {
			if peak.EffectSize <= 6 {
				continue
			}
			statusId := fmt.Sprintf("%v-%v", pmc, peak.PeakChannel)

			if peak.GlobalDifference > RoughnessItemThreshold {
				// It's roughness, can repeat so ensure we only save once
				if _, ok := roughnessPMCs[pmc]; !ok {
					status := "intensity-mismatch"
					if s, ok := peakStatuses.Statuses[statusId]; ok {
						status = s.Status
					}

					roughnessItems = append(roughnessItems, &protos.ClientRoughnessItem{
						Id:               int32(pmc),
						GlobalDifference: peak.GlobalDifference,
						Deleted:          status != "intensity-mismatch",
					})
					roughnessPMCs[pmc] = true
				}
			} else if peak.PeakHeight > 0.64 {
				startChannel := float32(peak.PeakChannel) - diffractionPeakHalfWidth
				endChannel := float32(peak.PeakChannel) + diffractionPeakHalfWidth

				channels := []float32{float32(peak.PeakChannel), startChannel, endChannel}
				keVs := []float64{}
				for det, cal := range spectrumEnergyCalibration.DetectorCalibrations {
					if det == DefaulteVCalibrationDetector {
						keVs = channelTokeV(channels, cal)
					}
				}

				if len(keVs) == 3 {
					status := "diffraction-peak"
					if s, ok := peakStatuses.Statuses[statusId]; ok {
						status = s.Status
					}

					allPeaks = append(allPeaks, &protos.ClientDiffractionPeak{
						Id: int32(pmc),
						Peak: &protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{
							PeakChannel:       peak.PeakChannel,
							EffectSize:        peak.EffectSize,
							BaselineVariation: peak.BaselineVariation,
							GlobalDifference:  peak.GlobalDifference,
							DifferenceSigma:   peak.DifferenceSigma,
							PeakHeight:        peak.PeakHeight,
							Detector:          peak.Detector,
						},
						EnergykeV:      float32(keVs[0]),
						StartEnergykeV: float32(keVs[1]),
						EndEnergykeV:   float32(keVs[2]),
						Status:         status,
					})
				}
			}
		}
	}

	return allPeaks, roughnessItems
}
