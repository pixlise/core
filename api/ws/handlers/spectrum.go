package wsHandler

import (
	"fmt"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleSpectrumReq(req *protos.SpectrumReq, hctx wsHelpers.HandlerContext) (*protos.SpectrumResp, error) {
	exprPB, startLocIdx, endLocIdx, err := beginDatasetFileReq(req.ScanId, req.StartingLocation, req.LocationCount, hctx)
	if err != nil {
		return nil, err
	}

	spectra := []*protos.Spectra{}

	for c := startLocIdx; c < endLocIdx; c++ {
		detectorSpectra := []*protos.Spectrum{}
		for _, detector := range exprPB.Locations[c].Detectors {
			detectorType := protos.SpectrumType_SPECTRUM_UNKNOWN
			detectorId := ""
			meta := map[int32]*protos.ScanMetaDataItem{}

			for _, m := range detector.Meta {
				if m.LabelIdx >= int32(len(exprPB.MetaLabels)) {
					return nil, fmt.Errorf("LabelIdx %v out of range when reading meta for spectrum at idx: %v", m.LabelIdx, c)
				}

				label := exprPB.MetaLabels[m.LabelIdx]
				if label == "DETECTOR_ID" {
					// Verify type
					if t := exprPB.MetaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
						detectorId = m.Svalue
					} else {
						return nil, fmt.Errorf("Unexpected %v when reading detector id for spectrum at idx: %v", t, c)
					}
				} else if label == "READTYPE" {
					// Verify type
					if t := exprPB.MetaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
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
						return nil, fmt.Errorf("Unexpected %v when reading spectrum type for spectrum at idx: %v", t, c)
					}
					// It's just meta-data to save, however, we don't need to save PMC in here!
				} else if label != "PMC" {
					// Check that this slot isn't taken
					if _, ok := meta[m.LabelIdx]; ok {
						return nil, fmt.Errorf("Conflicting label index %v when reading spectrum meta for spectrum at idx: %v", m.LabelIdx, c)
					}

					mSave := &protos.ScanMetaDataItem{}

					if t := exprPB.MetaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
						mSave.Value = &protos.ScanMetaDataItem_Svalue{Svalue: m.Svalue}
					} else if t == protos.Experiment_MT_INT {
						mSave.Value = &protos.ScanMetaDataItem_Ivalue{Ivalue: m.Ivalue}
					} else if t == protos.Experiment_MT_FLOAT {
						mSave.Value = &protos.ScanMetaDataItem_Fvalue{Fvalue: m.Fvalue}
					} else {
						return nil, fmt.Errorf("Unknown type %v for meta label: %v when reading spectrum type for spectrum at idx: %v", t, label, c)
					}

					meta[m.LabelIdx] = mSave
				}
			}

			// If we didn't get anything, complain
			if len(detectorId) <= 0 {
				return nil, fmt.Errorf("Failed to read detector id for spectrum at idx: %v", c)
			} else if detectorType == protos.SpectrumType_SPECTRUM_UNKNOWN {
				return nil, fmt.Errorf("Failed to read spectrum type for spectrum at idx: %v", c)
			}

			spectrum := &protos.Spectrum{
				Detector: detectorId,
				Type:     detectorType,
				Meta:     meta,
				MaxCount: int64(detector.SpectrumMax),
				Counts:   utils.ConvertIntSlice[int64](detector.Spectrum),
			}

			detectorSpectra = append(detectorSpectra, spectrum)
		}
		spectra = append(spectra, &protos.Spectra{
			Spectra: detectorSpectra,
		})
	}

	return &protos.SpectrumResp{
		SpectraPerLocation: spectra,
	}, nil
}
