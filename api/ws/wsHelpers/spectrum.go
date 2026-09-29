package wsHelpers

import (
	"fmt"
	"slices"
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

func MergeSpectra(exprPB *protos.Experiment, uploadedSpectra []*protos.Spectra) error {
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
				newDet := &protos.Experiment_Location_DetectorSpectrum{
					SpectrumMax: int32(spectrum.MaxCount),
					Meta: []*protos.Experiment_Location_MetaDataItem{
						{
							LabelIdx: int32(detectorIdIdx),
							Svalue:   detToWrite,
						},
						{
							LabelIdx: int32(readtypeIdx),
							Ivalue:   int32(spectrum.Type),
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
