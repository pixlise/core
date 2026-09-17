package bruker

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"

	brukerreader "github.com/pixlise/core/v4/api/dataimport/internal/converters/bruker/bruker-reader"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func IsBrukerFormat(importPath string) bool {
	return len(getBCFFile(importPath)) > 0
}

type Bruker struct {
	Downsample int
	ReadType   brukerreader.SelectType
}

func (b Bruker) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	// Find the one bcf file (if exists)
	bcfPath := getBCFFile(importPath)
	if len(bcfPath) <= 0 {
		return nil, "", errors.New("Bruker BCF file not found")
	}

	// TODO: Figure out a way to save some more of the metadata in the BCF:
	// Position parameters like stage XYZ, detector serial/window, energy characteristics, calibration params,
	// date/timestamps, etc. Most of these were catered for in the PIXLISE world by having a detector config
	// but these files come with all this so we probably need to not rely on an individual config but have
	// both options available?
	data := &dataConvertModels.OutputData{
		ClearBeforeSave: true,
		DatasetID:       datasetIDExpected,
		Instrument:      protos.ScanInstrument_BRUKER,
		Meta:            dataConvertModels.FileMetaData{Title: ""},
		//DetectorConfigFull:   detector,
		PseudoRanges:         []dataConvertModels.PseudoIntensityRange{},
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		MatchedAlignedImages: []dataConvertModels.MatchedAlignedImageMeta{},
		CreatorUserId:        sessionuser.PIXLISESystemUserId, // TODO: set a real creator
	}

	var spectrumWidth, spectrumHeight int

	beamLookup := dataConvertModels.BeamLocationByPMC{}
	locSpectraLookup := dataConvertModels.DetectorSampleByPMC{}

	// Read spectra first so we know what image to match to them
	spectrumItems, err := brukerreader.FileReader(bcfPath, brukerreader.SelectSpectrumImage, 0, false, b.Downsample, brukerreader.CutoffAuto, 0, "", true)
	for _, item := range spectrumItems {
		signal := ""
		signal, data.Meta.Title, err = readBasicItemMeta(item)
		if err != nil {
			return nil, "", err
		}

		if signal != "spectrum" {
			return nil, "", fmt.Errorf("Expected signal spectrum, got %v", signal)
		}

		// Write out image PMCs
		data.DetectorConfigFull, beamLookup, locSpectraLookup, spectrumWidth, spectrumHeight, err = b.readSpectra(item)
		if err != nil {
			return nil, "", err
		}
	}

	// Now we read all images
	// If we find an image to associate with the scan, do it and put it in PMC 1 (so it matches the beam locations)
	// All other images can be "matched-aligned" images to that one, so we piggyback on the beam locations used for PMC 1
	// but we can scale/move the image to be relative to the first one
	imageItems, err := brukerreader.FileReader(bcfPath, brukerreader.SelectImage, 0, false, b.Downsample, brukerreader.CutoffAuto, 0, "", true)
	if err != nil {
		return nil, "", err
	}

	// Ensure all images have the expected signal set
	for _, item := range imageItems {
		signal, _, err := readBasicItemMeta(item)
		if err != nil {
			return nil, "", err
		}

		if signal != "image" {
			return nil, "", fmt.Errorf("Expected signal image, got %v", signal)
		}
	}

	// First, loop through and find the spectrum-related image
	var spectrumImage *brukerreader.HspyItem
	var otherImages []*brukerreader.HspyItem

	for _, item := range imageItems {
		if spectrumImage == nil {
			axes, err := readAxes(item.Axes)
			if err != nil {
				return nil, "", fmt.Errorf("Failed to read image axes %v", err)
			}

			// We have to account for downsampling :( It might not be an exact multiple
			if math.Abs(float64(axes.width-spectrumWidth*b.Downsample)) < float64(b.Downsample)/2 &&
				math.Abs(float64(axes.height-spectrumHeight*b.Downsample)) < float64(b.Downsample)/2 {
				// Found a match, stop here
				spectrumImage = item

				// From here use the image size as spectrum size for any comparisons coming...
				spectrumWidth = axes.width
				spectrumHeight = axes.height
			}
		}

		if item != spectrumImage {
			otherImages = append(otherImages, item)
		}
	}

	// If we have a spectrum image, read it and set up as the default image aligned with beam locations
	contextImgsPerPMC := map[int32]string{}

	if spectrumImage != nil {
		imgPath, _ /*bytesPerChannel*/, width, height, err := b.readImage("spectra", importPath, spectrumImage)
		if err != nil {
			return nil, "", err
		}

		// Redundant check again...
		if width != spectrumWidth || height != spectrumHeight {
			return nil, "", fmt.Errorf("Spectrum image size %v x %v mismatch with spectrum data cube %v x %v", width, height, spectrumWidth, spectrumHeight)
		}

		// Set this as the default, first image
		contextImgsPerPMC[1] = filepath.Base(imgPath)
	}

	// We generate new PMCs for subsequent images - so find the max PMC we have so far here
	maxPMC := int32(1)
	for pmc := range locSpectraLookup {
		if pmc > maxPMC {
			maxPMC = pmc
		}
	}

	// Loop through the "other" images - see if we can form some RGB images out of them along the way too
	if alignedImgs, err := b.readAlignedImages(datasetIDExpected, otherImages, importPath, spectrumWidth, spectrumHeight, maxPMC); err != nil {
		return nil, "", err
	} else {
		data.MatchedAlignedImages = alignedImgs
	}

	data.SetPMCData(beamLookup, dataConvertModels.HousekeepingData{}, locSpectraLookup, contextImgsPerPMC, dataConvertModels.PseudoIntensities{}, map[int32]string{})

	return data, importPath, nil
}
