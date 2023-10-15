package wsHandler

import (
	"errors"
	"fmt"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleSpectrumReq(req *protos.SpectrumReq, hctx wsHelpers.HandlerContext) (*protos.SpectrumResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	detectorIdIdx := -1
	readtypeIdx := -1
	for c, label := range exprPB.MetaLabels {
		if label == "DETECTOR_ID" {
			detectorIdIdx = c
		} else if label == "READTYPE" {
			readtypeIdx = c
		}

		if readtypeIdx > -1 && detectorIdIdx > -1 {
			break
		}
	}

	spectra := []*protos.Spectra{}

	for _, c := range indexes {
		detectorSpectra := []*protos.Spectrum{}
		for _, detector := range exprPB.Locations[c].Detectors {
			spectrum, err := convertSpectrum(detector, exprPB.MetaLabels, exprPB.MetaTypes, detectorIdIdx, readtypeIdx)
			if err != nil {
				return nil, fmt.Errorf("%v for spectrum at idx: %v", err, c)
			}

			detectorSpectra = append(detectorSpectra, spectrum)
		}
		spectra = append(spectra, &protos.Spectra{Spectra: detectorSpectra})
	}

	// For now, all our spectra are 4096 but in time if we have other detectors, we should be able to send back
	// the channel count based on the detector here...
	channelCount := uint32(4096)

	result := &protos.SpectrumResp{
		SpectraPerLocation:   spectra,
		ChannelCount:         channelCount,
		NormalSpectraForScan: uint32(exprPB.NormalSpectra),
		DwellSpectraForScan:  uint32(exprPB.DwellSpectra),
	}

	// If bulk or max spectra are required... find & return those too
	if req.BulkSum || req.MaxValue {
		bulkSpectra, maxSpectra, err := getBulkMaxSpectra(req.BulkSum, req.MaxValue, exprPB, detectorIdIdx, readtypeIdx)
		if err != nil {
			return nil, err
		}

		result.BulkSpectra = bulkSpectra
		result.MaxSpectra = maxSpectra
	}

	return result, nil
}

// Returns array of bulk, array of max, error
func getBulkMaxSpectra(getBulk bool, getMax bool,
	exprPB *protos.Experiment,
	detectorIdIdx int,
	readtypeIdx int) ([]*protos.Spectrum, []*protos.Spectrum, error) {
	bulkSpectra := []*protos.Spectrum{}
	maxSpectra := []*protos.Spectrum{}

	for c, loc := range exprPB.Locations {
		for _, detector := range loc.Detectors {
			for _, m := range detector.Meta {
				if m.LabelIdx == int32(readtypeIdx) {
					// Verify type
					if t := exprPB.MetaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
						// These are hard-coded string values
						switch m.Svalue {
						case "BulkSum":
							if getBulk {
								spectrum, err := convertSpectrum(detector, exprPB.MetaLabels, exprPB.MetaTypes, detectorIdIdx, readtypeIdx)
								if err != nil {
									return nil, nil, err
								}
								bulkSpectra = append(bulkSpectra, spectrum)
								break
							}
						case "MaxValue":
							if getMax {
								spectrum, err := convertSpectrum(detector, exprPB.MetaLabels, exprPB.MetaTypes, detectorIdIdx, readtypeIdx)
								if err != nil {
									return nil, nil, err
								}
								maxSpectra = append(maxSpectra, spectrum)
								break
							}
						}
					} else {
						return nil, nil, fmt.Errorf("Unexpected %v when reading spectrum type for spectrum at idx: %v", t, c)
					}
				}
			}
		}

		// At this point, if we've what we're after, stop. We assume that ALL detectors bulk or max spectra will only be defined
		// in a single location.
		if (!getBulk || getBulk && len(bulkSpectra) > 0) &&
			(!getMax || getMax && len(maxSpectra) > 0) {
			break
		}
	}

	return bulkSpectra, maxSpectra, nil
}

func convertSpectrum(
	detector *protos.Experiment_Location_DetectorSpectrum,
	metaLabels []string,
	metaTypes []protos.Experiment_MetaDataType,
	detectorIdIdx int,
	readtypeIdx int) (*protos.Spectrum, error) {
	detectorType := protos.SpectrumType_SPECTRUM_UNKNOWN
	detectorId := ""
	meta := map[int32]*protos.ScanMetaDataItem{}

	for _, m := range detector.Meta {
		if m.LabelIdx >= int32(len(metaLabels)) {
			return nil, fmt.Errorf("LabelIdx %v out of range when reading meta", m.LabelIdx)
		}

		label := metaLabels[m.LabelIdx]
		if m.LabelIdx == int32(detectorIdIdx) {
			// Verify type
			if t := metaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
				detectorId = m.Svalue
			} else {
				return nil, fmt.Errorf("Unexpected %v when reading detector id", t)
			}
		} else if m.LabelIdx == int32(readtypeIdx) {
			// Verify type
			if t := metaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
				// These are hard-coded string values
				switch m.Svalue {
				case "Normal":
					detectorType = protos.SpectrumType_SPECTRUM_NORMAL
				case "Dwell":
					detectorType = protos.SpectrumType_SPECTRUM_DWELL
				case "BulkSum":
					detectorType = protos.SpectrumType_SPECTRUM_BULK
				case "MaxValue":
					detectorType = protos.SpectrumType_SPECTRUM_MAX
				}
			} else {
				return nil, fmt.Errorf("Unexpected %v when reading spectrum type", t)
			}
			// It's just meta-data to save, however, we don't need to save PMC in here!
		} else if label != "PMC" {
			// Check that this slot isn't taken
			if _, ok := meta[m.LabelIdx]; ok {
				return nil, fmt.Errorf("Conflicting label index %v when reading spectrum meta", m.LabelIdx)
			}

			mSave := &protos.ScanMetaDataItem{}

			if t := metaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
				mSave.Value = &protos.ScanMetaDataItem_Svalue{Svalue: m.Svalue}
			} else if t == protos.Experiment_MT_INT {
				mSave.Value = &protos.ScanMetaDataItem_Ivalue{Ivalue: m.Ivalue}
			} else if t == protos.Experiment_MT_FLOAT {
				mSave.Value = &protos.ScanMetaDataItem_Fvalue{Fvalue: m.Fvalue}
			} else {
				return nil, fmt.Errorf("Unknown type %v for meta label: %v when reading spectrum type", t, label)
			}

			meta[m.LabelIdx] = mSave
		}
	}

	// If we didn't get anything, complain
	if len(detectorId) <= 0 {
		return nil, errors.New("Failed to read detector id")
	} else if detectorType == protos.SpectrumType_SPECTRUM_UNKNOWN {
		return nil, errors.New("Failed to read spectrum type")
	}

	spectrum := &protos.Spectrum{
		Detector: detectorId,
		Type:     detectorType,
		Meta:     meta,
		MaxCount: uint32(detector.SpectrumMax),
		Counts:   utils.ConvertIntSlice[uint32](detector.Spectrum),
	}

	return spectrum, nil
}
