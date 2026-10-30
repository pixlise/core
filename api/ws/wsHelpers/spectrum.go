package wsHelpers

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"google.golang.org/protobuf/proto"
)

// Here we merge existing scan data with spectra from the temp dir and the last downloaded set
// to form a new scan file that can be saved containing everything
// NOTE: We're just replacing the spectra in the passed in experiment data (that is expected to
//
//	be modifiable as it was likely just loaded from a file anyway!)
func ReadSpectraUploads(spectraTempFilesPath string, svcs *services.APIServices) ([]*protos.Spectra, error) {
	// Get a list of all temp files and load each
	spectraTmpFiles, err := svcs.FS.ListObjects(svcs.Config.DatasetsBucket, spectraTempFilesPath)
	if err != nil {
		return nil, err
	}

	slices.Sort(spectraTmpFiles)

	allSpectra := []*protos.Spectra{}
	for _, p := range spectraTmpFiles {
		d, err := svcs.FS.ReadObject(svcs.Config.DatasetsBucket, p)
		if err != nil {
			return nil, fmt.Errorf("Failed to read %v: %v", p, err)
		}

		req := &protos.SpectrumUploadReq{}
		err = proto.Unmarshal(d, req)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode %v: %v", p, err)
		}

		// Add to the list
		allSpectra = append(allSpectra, req.SpectraPerLocation...)
	}

	return allSpectra, nil
}

func MergeSpectra(exprPB *protos.Experiment, uploadedSpectra []*protos.Spectra, makeBulkSum bool) error {
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

	// If we're making a bulk sum too, find the index to add it to
	bulkSumLocIdx := -1
	if makeBulkSum {
		bulkSumLocIdx = findBulkSumLocIdx(exprPB, readtypeIdx)

		if bulkSumLocIdx < 0 {
			return errors.New("No location found for saving bulk sum")
		}

		bulkSums := makeBulkSums(uploadedSpectra, exprPB.MetaLabels)

		if len(bulkSums) <= 0 {
			return errors.New("No bulk sums calculated")
		}

		uploadedSpectra[bulkSumLocIdx].Spectra = append(uploadedSpectra[bulkSumLocIdx].Spectra, bulkSums...)
	}

	if detectorIdIdx == -1 || readtypeIdx == -1 {
		return fmt.Errorf("No label index found for DETECTOR_ID (%v) or READTYPE (%v)", detectorIdIdx, readtypeIdx)
	}

	if len(uploadedSpectra) != len(exprPB.Locations) {
		return fmt.Errorf("Expected %v spectra, got %v to merge", len(exprPB.Locations), len(uploadedSpectra))
	}

	// Run through each and merge with what's already in the scan
	for locIdx, loc := range exprPB.Locations {
		// Add all spectra we want to add to it, note that we don't allow overwriting original detectors
		// so we append a ' to the names of any incoming detectors
		for specIdx, spectrum := range uploadedSpectra[locIdx].Spectra {
			detToWrite := spectrum.Detector
			if len(detToWrite) < 1 {
				return fmt.Errorf("Detector not set for spectrum %v of location %v", specIdx, locIdx)
			}
			if len(detToWrite) == 1 {
				// Append a '
				detToWrite = detToWrite + "'"
			}

			// Check if there's an existing detector we can write this into
			detIdxToWrite := -1
			for detIdx, d := range loc.Detectors {
				// Get existing detector, check if it's one we're editing
				for _, m := range d.Meta {
					if m.LabelIdx == int32(detectorIdIdx) && m.Svalue == detToWrite {
						detIdxToWrite = detIdx
						break
					}
				}
			}

			// If no detector found that matches, we create a new one
			if detIdxToWrite == -1 {
				// Oh no! At some point we defined these with different integer types... shaaaame :(
				spectrumType := utils.SpectrumTypeToString(spectrum.Type)
				if len(spectrumType) <= 0 {
					spectrumType = "Normal"
				}

				newDet := &protos.Experiment_Location_DetectorSpectrum{
					SpectrumMax: int32(spectrum.MaxCount),
					Meta: []*protos.Experiment_Location_MetaDataItem{
						{
							LabelIdx: int32(detectorIdIdx),
							Svalue:   detToWrite,
						},
						{
							LabelIdx: int32(readtypeIdx),
							Svalue:   spectrumType,
						},
					},
				}

				// Set meta fields if any
				interestedFields := []string{"LIVETIME", "OFFSET", "REALTIME", "XPERCHAN"} // NOTE: DETECTOR_ID, READTYPE added already above
				addedFieldCount := 0
				for k, v := range spectrum.Meta {
					if k < int32(len(exprPB.MetaLabels)) && utils.ItemInSlice(exprPB.MetaLabels[k], interestedFields) {
						m := &protos.Experiment_Location_MetaDataItem{LabelIdx: k}
						if exprPB.MetaTypes[k] == protos.Experiment_MT_STRING {
							m.Svalue = v.GetSvalue()
						} else if exprPB.MetaTypes[k] == protos.Experiment_MT_FLOAT {
							m.Fvalue = v.GetFvalue()
						} else if exprPB.MetaTypes[k] == protos.Experiment_MT_INT {
							m.Ivalue = v.GetIvalue()
						}

						newDet.Meta = append(newDet.Meta, m)
						addedFieldCount++
					}
				}

				// Ensure all meta fields we wanted were specified
				if addedFieldCount != len(interestedFields) {
					return fmt.Errorf("Index %v spectrum %v was missing one of the following meta fields: %v", locIdx, specIdx, strings.Join(interestedFields, ","))
				}

				for _, c := range spectrum.Counts {
					newDet.Spectrum = append(newDet.Spectrum, int32(c))
				}

				loc.Detectors = append(loc.Detectors, newDet)
			} else {
				counts := []int32{}
				for _, c := range spectrum.Counts {
					counts = append(counts, int32(c))
				}

				loc.Detectors[detIdxToWrite].Spectrum = counts
				loc.Detectors[detIdxToWrite].SpectrumMax = int32(spectrum.MaxCount)
			}
		}
	}

	return nil
}

func findBulkSumLocIdx(exprPB *protos.Experiment, readtypeIdx int) int {
	for locIdx, loc := range exprPB.Locations {
		for _, det := range loc.Detectors {
			for _, m := range det.Meta {
				if m.LabelIdx == int32(readtypeIdx) && m.Svalue == "BulkSum" {
					return locIdx
				}
			}
		}
	}

	return -1
}

func makeBulkSums(uploadedSpectra []*protos.Spectra, metaLabels []string) []*protos.Spectrum {
	// Find the indexes for what we're interested in
	labels := []string{"LIVETIME", "REALTIME", "OFFSET", "XPERCHAN"}
	toAverage := []string{"OFFSET", "XPERCHAN"}
	labelIndexes := map[string]int32{}
	for i, l := range metaLabels {
		if utils.ItemInSlice(l, labels) {
			labelIndexes[l] = int32(i)
		}
	}

	bulkSums := map[string]*protos.Spectrum{}
	n := float32(0)
	detectorToCount := ""
	for _ /*locIdx*/, spectra := range uploadedSpectra {
		for _, s := range spectra.Spectra {
			if len(detectorToCount) <= 0 {
				detectorToCount = s.Detector
			}

			bs := bulkSums[s.Detector]

			countsDecoded := utils.ZeroRunDecode(s.Counts)
			if bs == nil {
				bs = &protos.Spectrum{
					Detector: s.Detector,
					Type:     protos.SpectrumType_SPECTRUM_BULK,
					Counts:   make([]uint32, len(countsDecoded)),
					MaxCount: 0,
					Meta:     map[int32]*protos.ScanMetaDataItem{},
				}
				bulkSums[s.Detector] = bs
			}

			if s.Type == protos.SpectrumType_SPECTRUM_NORMAL {
				// Add spectrum to the one we already have
				for i, count := range countsDecoded {
					bs.Counts[i] += count
				}

				// Fill out the meta data
				for _ /*label*/, idx := range labelIndexes {
					if idx >= 0 {
						prev := float32(0)
						if v, ok := bs.Meta[idx]; ok {
							prev = v.GetFvalue()
						} else {
							bs.Meta[idx] = &protos.ScanMetaDataItem{}
						}

						bs.Meta[idx].Value = &protos.ScanMetaDataItem_Fvalue{
							Fvalue: prev + s.Meta[idx].GetFvalue(),
						}
					}
				}

				// Count it if we have the right detector
				if s.Detector == detectorToCount {
					n++
				}
			}
		}
	}

	// Save them sorted by detector so our tests can deterministic order
	detectors := utils.GetMapKeys(bulkSums)
	sort.Strings(detectors)

	result := []*protos.Spectrum{}

	for _, det := range detectors {
		bs := bulkSums[det]

		// Update max counts
		for _, c := range bs.Counts {
			if c > bs.MaxCount {
				bs.MaxCount = c
			}
		}

		// Encode for efficiency again
		bs.Counts = utils.ZeroRunEncode(bs.Counts)

		// Fix meta values that need to be averages
		for _, l := range toAverage {
			if i, ok := labelIndexes[l]; ok {
				bs.Meta[i].Value = &protos.ScanMetaDataItem_Fvalue{
					Fvalue: bs.Meta[i].GetFvalue() / n,
				}
			}
		}

		// Put them in a list
		result = append(result, bs)
	}

	return result
}

func GenerateAggregateSpectrum(
	indexes []uint32, detectors []string, readType protos.SpectrumType,
	genBulk bool, genMax bool,
	exprPB *protos.Experiment) (map[string]*protos.Spectrum, map[string]*protos.Spectrum, error) {
	// check for insanity
	if !genBulk && !genMax {
		return nil, nil, errors.New("Requested neither bulk sum nor max value")
	}

	// Ensure there's a detector
	if len(detectors) <= 0 {
		return nil, nil, errors.New("Request didn't specify detector(s)")
	}

	detectorsRequested := map[string]bool{}
	for _, det := range detectors {
		detectorsRequested[det] = true
	}

	if readType == protos.SpectrumType_SPECTRUM_UNKNOWN {
		return nil, nil, errors.New("Request didn't specify spectrum type")
	}

	interestingLabels := []string{"DETECTOR_ID", "READTYPE", "LIVETIME", "REALTIME", "OFFSET", "XPERCHAN"}
	labelIdxs := map[string]int{}

	for c, label := range exprPB.MetaLabels {
		if utils.ItemInSlice(label, interestingLabels) {
			labelIdxs[label] = c
		}

		if len(labelIdxs) == len(interestingLabels) {
			break
		}
	}

	if len(labelIdxs) != len(interestingLabels) {
		return nil, nil, fmt.Errorf("Missing meta labels from spectra. Need [%v], found [%v]", interestingLabels, utils.GetMapKeys(labelIdxs))
	}

	detectorIdMetaIdx := -1
	readtypeMetaIdx := -1

	bulkSums := map[string]*protos.Spectrum{}
	maxValues := map[string]*protos.Spectrum{}

	readTypeRequestedStr := utils.SpectrumTypeToString(readType)
	if len(readTypeRequestedStr) <= 0 {
		return nil, nil, fmt.Errorf("Requested invalid spectrum type %v", readType)
	}

	for _, locIdx := range indexes {
		if locIdx >= uint32(len(exprPB.Locations)) {
			return nil, nil, fmt.Errorf("Invalid location %v, location count is %v", locIdx, len(exprPB.Locations))
		}

		// Loop through all detectors and read the ones we're interested in. If we requested detectors that don't exist, fail!
		detectorsFound := map[string]bool{}
		for _, d := range detectors {
			detectorsFound[d] = false
		}

		for _, det := range exprPB.Locations[locIdx].Detectors {
			thisDetectorId, thisReadType, err := getDetectorMetaValues(det, int32(labelIdxs["DETECTOR_ID"]), int32(labelIdxs["READTYPE"]), &detectorIdMetaIdx, &readtypeMetaIdx)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed reading location %v: %v", locIdx, err)
			}

			// Is this one of the detectors/read types we care about
			if !detectorsRequested[thisDetectorId] || thisReadType != readTypeRequestedStr {
				// Nope, ignore it
				continue
			}

			// Remember we found this one
			detectorsFound[thisDetectorId] = true

			// OK we want to add this to our outputs, decode its spectrum
			// NOTE: Why this is so complex - we want to keep it lean for requesting just an individual spectrum!
			//       so we try to minimise decoding, running through the counts array and re-encoding at the end
			//       for simpler cases. The test checks for single and multi index cases to ensure we're still
			//       returning accurate results
			countsDecoded := []int32{}
			if len(indexes) > 1 {
				// Only decode if we're doing any processing on these
				countsDecoded = utils.ZeroRunDecode(det.Spectrum)
			}

			// Process bulk sum if needed
			if genBulk {
				bs := bulkSums[thisDetectorId]
				if bs == nil {
					bs = makeSpectrumCopy(det, thisDetectorId, readType, len(countsDecoded), makeSpectrumMeta(labelIdxs))
					bulkSums[thisDetectorId] = bs
				}

				if len(countsDecoded) > 0 {
					// Add spectrum to the one we already have
					for i, count := range countsDecoded {
						bs.Counts[i] += uint32(count)
					}
				}

				updateSpectrumMeta(bs, det.Meta)
			}

			// Process max value if needed
			if genMax {
				bs := maxValues[thisDetectorId]
				if bs == nil {
					bs = makeSpectrumCopy(det, thisDetectorId, readType, len(countsDecoded), makeSpectrumMeta(labelIdxs))
					maxValues[thisDetectorId] = bs
				}

				if len(countsDecoded) > 0 {
					// Max spectrum to the one we already have
					for i, count := range countsDecoded {
						if uint32(count) > bs.Counts[i] {
							bs.Counts[i] = uint32(count)
						}
					}
				}

				updateSpectrumMeta(bs, det.Meta)
			}
		}

		// Did we miss any detectors?
		for d, found := range detectorsFound {
			if !found {
				return nil, nil, fmt.Errorf("Requested detector %v, read type %v not found at location %v", d, readTypeRequestedStr, locIdx)
			}
		}
	}

	if len(indexes) > 1 {
		// Update the max counts for all, then compress each spectrum
		x := []map[string]*protos.Spectrum{bulkSums, maxValues}
		calibrationLabelIdxs := []int32{int32(labelIdxs["OFFSET"]), int32(labelIdxs["XPERCHAN"])}
		for isMax, bmspectra := range x {
			for _, s := range bmspectra {
				for _, count := range s.Counts {
					if count > s.MaxCount {
						s.MaxCount = count
					}
				}
				s.Counts = utils.ZeroRunEncode(s.Counts)

				// Also modify the meta values as needed
				// Livetime and Realtime are averaged for max spectra
				// Offset and XperChan are averaged for all spectra
				for idx, v := range s.Meta {
					if utils.ItemInSlice(idx, calibrationLabelIdxs) || isMax == 1 {
						if x, ok := v.Value.(*protos.ScanMetaDataItem_Fvalue); ok {
							x.Fvalue = x.Fvalue / float32(len(indexes))
						}
					}
				}
			}
		}
	}

	return bulkSums, maxValues, nil
}

func makeSpectrumCopy(
	det *protos.Experiment_Location_DetectorSpectrum,
	detectorId string, readType protos.SpectrumType,
	decodedCount int,
	meta map[int32]*protos.ScanMetaDataItem) *protos.Spectrum {
	result := &protos.Spectrum{
		Detector: detectorId,
		Type:     readType,
		MaxCount: uint32(det.SpectrumMax),
		Meta:     meta,
	}

	if decodedCount <= 0 {
		// We're simply copying this spectrum and sending it
		result.Counts = make([]uint32, len(det.Spectrum))
		for i, c := range det.Spectrum {
			result.Counts[i] = uint32(c)
		}
	} else {
		// Decode it to allow further processing
		result.Counts = make([]uint32, decodedCount)
	}

	return result
}

func makeSpectrumMeta(labelIdxs map[string]int) map[int32]*protos.ScanMetaDataItem {
	return map[int32]*protos.ScanMetaDataItem{
		int32(labelIdxs["OFFSET"]):   {Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 0}},
		int32(labelIdxs["XPERCHAN"]): {Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 0}},
		int32(labelIdxs["REALTIME"]): {Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 0}},
		int32(labelIdxs["LIVETIME"]): {Value: &protos.ScanMetaDataItem_Fvalue{Fvalue: 0}},
	}
}

func updateSpectrumMeta(toUpdate *protos.Spectrum, updateWithMeta []*protos.Experiment_Location_MetaDataItem) {
	// Add to each value that we care about
	for idx, f := range toUpdate.Meta {
		// Find the corresponding field in the incoming spectrum
		for _, m := range updateWithMeta {
			if m.LabelIdx == idx {
				if x, ok := f.Value.(*protos.ScanMetaDataItem_Fvalue); ok {
					x.Fvalue = x.Fvalue + m.Fvalue
				}
				break
			}
		}
	}
}

func getDetectorMetaValues(
	det *protos.Experiment_Location_DetectorSpectrum,
	detectorIdLabelIdx int32, readTypeLabelIdx int32,
	lastDetectorIdMetaIdx *int, lastReadTypeMetaIdx *int) (string, string, error) {
	// Check if the ones at the same meta index are valid... if not we search from the start
	thisDetectorId := ""
	thisReadType := ""

	if *lastDetectorIdMetaIdx >= 0 && *lastDetectorIdMetaIdx < len(det.Meta) && det.Meta[*lastDetectorIdMetaIdx].LabelIdx == detectorIdLabelIdx {
		// It's in the same spot so just grab it
		thisDetectorId = det.Meta[*lastDetectorIdMetaIdx].Svalue
	}

	if *lastReadTypeMetaIdx >= 0 && *lastReadTypeMetaIdx < len(det.Meta) && det.Meta[*lastReadTypeMetaIdx].LabelIdx == readTypeLabelIdx {
		thisReadType = det.Meta[*lastReadTypeMetaIdx].Svalue
	}

	if len(thisDetectorId) <= 0 || len(thisReadType) <= 0 {
		// We don't have a valid one of either of these, so search!
		for i, m := range det.Meta {
			if m.LabelIdx == int32(detectorIdLabelIdx) {
				thisDetectorId = m.Svalue
				*lastDetectorIdMetaIdx = i // Remember for subsequent call
			}

			if m.LabelIdx == int32(readTypeLabelIdx) {
				thisReadType = m.Svalue
				*lastReadTypeMetaIdx = i // Remember for subsequent call
			}

			if len(thisDetectorId) > 0 && len(thisReadType) > 0 {
				break // don't loop if we've found all we need
			}
		}
	}

	// At this point if either is empty, stop
	if len(thisDetectorId) <= 0 {
		return thisDetectorId, thisReadType, errors.New("detector id not found")
	}
	if len(thisReadType) <= 0 {
		return thisDetectorId, thisReadType, errors.New("spectrum read type not found")
	}

	return thisDetectorId, thisReadType, nil
}
