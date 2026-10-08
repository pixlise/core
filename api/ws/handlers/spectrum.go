package wsHandler

import (
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
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
		TimeStampUnixSec:     uint32(exprPB.ImportTimeStampUnixSec),
		SpectraPerLocation:   spectra,
		ChannelCount:         channelCount,
		NormalSpectraForScan: uint32(exprPB.NormalSpectra),
		DwellSpectraForScan:  uint32(exprPB.DwellSpectra),
	}

	for c, label := range exprPB.MetaLabels {
		if label == "LIVETIME" {
			result.LiveTimeMetaIndex = uint32(c)
			break
		}
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

func HandleSpectrumUploadReq(req *protos.SpectrumUploadReq, hctx wsHelpers.HandlerContext) (*protos.SpectrumUploadResp, error) {
	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	// Lets check that spectra uploaded are:
	// - The same size as existing
	// - For valid existing location indexes
	// - Not overwriting existing detector spectra. Eg if there's an "A" spectrum, new upload can't be "A", we append
	//   a suffix so save as "A'". We do allow overwriting a suffixed detectors spectrum though, so "A'" can be edited
	// We save each upload in a separate file in S3 where our scan is stored (in a separate sub-dir) until the last spectra
	// arrive, at which point we merge all the new spectra into the dataset.bin file.
	// Longer-term if we have huge data-sets this won't be a valid way to operate but for operating with PIXL data this
	// should suffice for now. Re-organising for longer-term storage will require other re-architecting anyway.

	// If it's not the last, we just save
	spectraUploadSubdir := "spectra-upload"

	lastLocUploaded := int(req.FirstLocationIndex) + len(req.SpectraPerLocation)
	if lastLocUploaded < len(exprPB.Locations) {
		// Just save the file
		fileName := fmt.Sprintf("%v/%06d.bin", spectraUploadSubdir, req.FirstLocationIndex)
		s3Path := filepaths.GetScanFilePath(req.ScanId, fileName)

		reqBytes, err := proto.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("Failed to serialise spectrum upload file %v. Error: %v", s3Path, err)
		}

		hctx.Svcs.Log.Debugf("Writing temp spectra upload file: s3://%v/%v", hctx.Svcs.Config.DatasetsBucket, s3Path)
		err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.DatasetsBucket, s3Path, reqBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to write spectrum upload file %v. Error: %v", s3Path, err)
		}

		// Otherwise we've written it, yay!
	} else {
		// ELSE: It looks like this was the last upload, so process it all and merge with existing spectra in the file
		s3Path := filepaths.GetScanFilePath(req.ScanId, spectraUploadSubdir)

		allSpectra, err := wsHelpers.ReadSpectraUploads(s3Path, hctx.Svcs)
		if err != nil {
			return nil, fmt.Errorf("Failed to read uploaded spectra: %v", err)
		}

		// Finish off with the batch
		if req.FirstLocationIndex != uint32(len(allSpectra)) {
			return nil, fmt.Errorf("Last spectrum list start index expected to be %v, got %v", len(allSpectra), req.FirstLocationIndex)
		}

		allSpectra = append(allSpectra, req.SpectraPerLocation...)

		if err = wsHelpers.MergeSpectra(exprPB, allSpectra); err != nil {
			return nil, fmt.Errorf("Failed merge uploaded spectra with existing scan data: %v", err)
		}

		// Overwrite the original scan data file
		if exprData, err := proto.Marshal(exprPB); err != nil {
			return nil, fmt.Errorf("Failed to serialise scan data: %v", err)
		} else {
			s3Path = filepaths.GetScanFilePath(req.ScanId, filepaths.DatasetFileName)
			err = hctx.Svcs.FS.WriteObject(hctx.Svcs.Config.DatasetsBucket, s3Path, exprData)

			if err != nil {
				return nil, fmt.Errorf("Failed to write modified scan file: %v", err)
			}

			// NOTE: At this point we DON'T need to re-run the diffraction detector - it was all about A & B, but we now
			//       have some undefined user-edited set of spectra with A & B unchanged. Diffraction DB is still valid!
			// TODO: At this point we could delete the uploaded spectra dir - only keeping it around for now for debugging purposes!
		}
	}

	// Success!
	return &protos.SpectrumUploadResp{}, nil
}
