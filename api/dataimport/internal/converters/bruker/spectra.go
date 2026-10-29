package bruker

import (
	"fmt"

	brukerreader "github.com/pixlise/core/v4/api/dataimport/internal/converters/bruker/bruker-reader"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Reads all spectra for an "image"
// Returns:
// - detector config we need to save
// - beam locations
// - spectra
// - width
// - height
// - error (or nil)
func (b Bruker) readSpectra(item *brukerreader.HspyItem) (
	*protos.DetectorConfig, dataConvertModels.BeamLocationByPMC, dataConvertModels.DetectorSampleByPMC, int, int, error) {
	// This is a "spectrum" file, read it in like we have PMCs! PMCs will be allocated by traversing the grid
	// and counting up, so we don't need to actually create a list here

	// w := fmt.Sprintf("Data type is: %T, %v|%v, channels %v\n", item.Data, zOffset, zScale, channels)
	// fmt.Println(w)

	beamLookup := dataConvertModels.BeamLocationByPMC{}
	//bulkMaxSpectraLookup := dataConvertModels.DetectorSampleByPMC{}
	locSpectraLookup := dataConvertModels.DetectorSampleByPMC{}

	axes, err := readAxes(item.Axes)
	if err != nil {
		return nil, beamLookup, locSpectraLookup, 0, 0, err
	}

	detector, err := readDetectorConfig(item)
	if err != nil {
		return nil, beamLookup, locSpectraLookup, 0, 0, err
	}

	cubeFunc, ok := item.Data.(brukerreader.LazyHyperCube)
	if !ok {
		return nil, beamLookup, locSpectraLookup, 0, 0, fmt.Errorf("Unexpected data type %T for spectrum data", cubeFunc)
	}

	cube, err := cubeFunc()
	if err != nil {
		return nil, beamLookup, locSpectraLookup, 0, 0, fmt.Errorf("Failed when reading spectrum data: %v", err)
	}

	// Confirm expected sizes
	if cube.Shape[0] != axes.height || cube.Shape[1] != axes.width || cube.Shape[2] != axes.channels {
		return nil, beamLookup, locSpectraLookup, 0, 0,
			fmt.Errorf("Unexpected difference with spectrum data shape %v,%v,%v not matching width %v, height %v, channels %v",
				cube.Shape[0], cube.Shape[1], cube.Shape[2], axes.width, axes.height, axes.channels)
	}

	fmt.Printf("Apply channel scale %v, offset %v\n", axes.zScale, axes.zOffset)

	stagePos := []float64{0, 0, 0}
	// for c, axis := range []string{"X", "Y", "Z"} {
	// 	if f, err := getMetaField(item.OriginalMetadata, "Stage", axis); err == nil {
	// 		stagePos[c] = f.(float64)
	// 	}
	// }

	bulkSpectrum := make([]int64, axes.channels)
	maxSpectrum := make([]int64, axes.channels)
	bulkLiveTime := float32(0)

	xPhysicalSize := axes.xScale
	if axes.units == "µm" {
		xPhysicalSize *= 0.001
	}
	yPhysicalSize := axes.yScale
	if axes.units == "µm" {
		yPhysicalSize *= 0.001
	}

	// Generate coordinates and read spectra
	for y := 0; y < axes.height; y++ {
		for x := 0; x < axes.width; x++ {
			pmc := int32(y*axes.width + x)

			// This is probably wrong... but some combination of this needs to be applied to the
			// x or y direction counter to find the physical movement, and perhaps the stage
			// xyz to start from...
			xPos := float32(stagePos[0]) + float32(axes.xOffset) + float32(x)*float32(xPhysicalSize)
			yPos := float32(stagePos[1]) + float32(axes.yOffset) + float32(y)*float32(yPhysicalSize)

			loc := dataConvertModels.BeamLocation{
				X:        xPos,
				Y:        yPos,
				Z:        float32(stagePos[2]),
				GeomCorr: 0,
				IJ:       map[int32]dataConvertModels.BeamLocationProj{},
			}

			// Add a coordinate for an image connected to PMC 1
			// Once we're read in the image reading code will need to ensure the right
			// image (with same dimensions etc as this one) is slotted into PMC 1!
			loc.IJ[1] = dataConvertModels.BeamLocationProj{
				I: float32(x * b.Downsample),
				J: float32(y * b.Downsample),
			}

			beamLookup[pmc+1] = loc

			cubeIdx := pmc * int32(axes.channels)
			readCounts := cube.Data[cubeIdx : cubeIdx+int32(axes.channels)]

			spectrumCounts := make([]int64, axes.channels)
			for i, c := range readCounts {
				spectrumCounts[i] = int64(c)
				bulkSpectrum[i] += int64(c)
				if int64(c) > maxSpectrum[i] {
					maxSpectrum[i] = int64(c)
				}
			}

			locSpectraLookup[pmc+1] = []dataConvertModels.DetectorSample{{
				Meta:     makeSpectrumMeta(1, 1, axes.channels, float32(axes.zOffset), float32(axes.zScale), "Normal", xPos, yPos, float32(stagePos[2])),
				Spectrum: spectrumCounts,
			}}
			bulkLiveTime += 1
		}
	}

	// Save a bulk spectrum for the first PMC
	locSpectraLookup[1] = append(locSpectraLookup[1], dataConvertModels.DetectorSample{
		Meta:     makeSpectrumMeta(bulkLiveTime, bulkLiveTime, axes.channels, float32(axes.zOffset), float32(axes.zScale), "BulkSum", float32(stagePos[0]), float32(stagePos[1]), float32(stagePos[2])),
		Spectrum: bulkSpectrum,
	})
	// And max!
	avgLiveTime := bulkLiveTime / float32(axes.width*axes.height)
	locSpectraLookup[1] = append(locSpectraLookup[1], dataConvertModels.DetectorSample{
		Meta:     makeSpectrumMeta(avgLiveTime, avgLiveTime, axes.channels, float32(axes.zOffset), float32(axes.zScale), "MaxValue", float32(stagePos[0]), float32(stagePos[1]), float32(stagePos[2])),
		Spectrum: maxSpectrum,
	})

	return detector, beamLookup, locSpectraLookup, axes.width, axes.height, nil
}

func makeSpectrumMeta(livetime, realtime float32, channels int, offset float32, xperchan float32, readType string, x, y, z float32) dataConvertModels.MetaData {
	return dataConvertModels.MetaData{
		// These are hard-coded because we haven't found a field to map them to in the incoming .bcf data
		"READTYPE":   dataConvertModels.StringMetaValue(readType),
		"SIGNALTYPE": dataConvertModels.StringMetaValue("XRF"),
		"LIVETIME":   dataConvertModels.FloatMetaValue(livetime),
		"REALTIME":   dataConvertModels.FloatMetaValue(realtime),

		"NPOINTS": dataConvertModels.StringMetaValue(fmt.Sprintf("%v", channels)),

		"XPERCHAN": dataConvertModels.FloatMetaValue(xperchan), //StringMetaValue(fmt.Sprintf("%v, %v", xperchan, xperchan)),
		"OFFSET":   dataConvertModels.FloatMetaValue(offset),   //StringMetaValue(fmt.Sprintf("%v, %v", offset, offset)),

		// Detector might need to be looked up in OriginalMetadata.Hardware.SelectedDetectors ??
		// The one example we have has DetectorCount = SelectedDetectors = 1 so it doesn't help!
		// Or perhaps LineCounter has something to do with it?
		"DETECTOR_ID": dataConvertModels.StringMetaValue("A"),
		"NCOLUMNS":    dataConvertModels.StringMetaValue("1"),
		"DATATYPE":    dataConvertModels.StringMetaValue("Y"),

		"XPOSITION": dataConvertModels.FloatMetaValue(x),
		"YPOSITION": dataConvertModels.FloatMetaValue(y),
		"ZPOSITION": dataConvertModels.FloatMetaValue(z),

		// "XUNITS":     dataConvertModels.StringMetaValue("eV"),
		// "YUNITS":     dataConvertModels.StringMetaValue("COUNTS"),

		// Pulse shaping and peaking time possibly defined in OriginalMetdata.Hardware
		// More pulse stuff in OriginalMetdata.Detector eg FWHMShift, PulsePairResTimeCount
		// ZeroPeakPosition=96 might be the "0th" channel?
	}
}

func readDetectorConfig(from *brukerreader.HspyItem) (*protos.DetectorConfig, error) {
	detector := &protos.DetectorConfig{
		// These are guesses - we don't have these in .bcf metadata fields...
		MinElement:      3,
		MaxElement:      92,
		XrfeVLowerBound: 200,
		// XRFeVUpperBound => OriginalMetadata.Analysis.PrimaryEnergy * 1000 ?
		XrfeVResolution: 145,  // From M4 tornado brochure it's 145eV
		MmBeamRadius:    0.02, // From M4 tornado brochure it's 20um
		// WindowElement => OriginalMetadata.WindowLayers, but this has 6 layers, elements 5,6,7,8,13,14 with thickest being 14 at "380" vs 0.235-0.013...
		// TubeElement => ??? OriginalMetadata.DetLayers, has 5 layers, elements 13,7,5,6,8 with thickest being 7 at 8.5E-2
	}

	var f interface{}
	var err error

	if f, err = getMetaField(from.Metadata, "Acquisition_instrument", "SEM", "Detector", "EDS", "elevation_angle"); err == nil {
		detector.ElevAngle = float32(f.(float64))
	} else {
		return nil, err
	}

	if f, err = getMetaField(from.Metadata, "Acquisition_instrument", "SEM", "Detector", "EDS", "detector_type"); err == nil {
		detector.Id = f.(string)
	} else {
		return nil, err
	}

	if f, err = getMetaField(from.OriginalMetadata, "Analysis", "PrimaryEnergy"); err == nil {
		detector.XrfeVUpperBound = int32(f.(int64) * 1000)
	} else {
		return nil, err
	}

	windowLayers := map[uint32]float64{}
	if f, err = getMetaField(from.OriginalMetadata, "Detector", "WindowLayers"); err == nil {
		layers := f.(map[string]interface{})
		for _, layerI := range layers {
			if atom, err := getMetaField(layerI.(map[string]interface{}), "Atom"); err == nil {
				if thickness, err := getMetaField(layerI.(map[string]interface{}), "Thickness"); err == nil {
					windowLayers[uint32(atom.(int64))] = thickness.(float64)
				} else {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	} else {
		return nil, err
	}

	maxThickness := float64(0)
	thickestAtom := uint32(0)
	for atom, thickness := range windowLayers {
		if thickness > maxThickness {
			maxThickness = thickness
			thickestAtom = atom
		}
	}

	if thickestAtom != 0 {
		detector.WindowElement = int32(thickestAtom)
	}

	return detector, nil
}
