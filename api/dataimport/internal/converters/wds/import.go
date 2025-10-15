package importwds

import (
	"fmt"
	"image"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"golang.org/x/image/tiff"
)

var suffixImageMap = ".map.tif"

func IsWDSMapFormat(importPath string) bool {
	localFS := &fileaccess.FSAccess{}

	files, err := localFS.ListObjects(importPath, "wds-data/")

	if err != nil {
		return false
	}

	// We only care about the wds-data directory, any other unzipped files we ignore...
	count := 0
	for _, file := range files {
		//if strings.HasPrefix(file, "wds-data/") {
		if strings.HasSuffix(file, suffixImageMap) {
			count++
		} else {
			return false
		}
		//}
	}

	return count > 0
}

type ImageMaps struct {
}

// For importing data from Yang Liu
// This importer expects TIF images with specific naming. Each image should be a single channel int or floating point image
// representing an element with each pixel named *_ElementSymbol_<number>.map.tif. There's a "combined" image, CP_<number>.map
// which can be considered the optical image and it will also be a mask to represent what parts of the element maps to ignore (black pixels)

func (im ImageMaps) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	localFS := &fileaccess.FSAccess{}

	files, err := localFS.ListObjects(importPath, "wds-data") // Allow any file name... previously was expecting to start with: datasetIDExpected+"_")

	if err != nil {
		return nil, "", err
	}

	// Allocate everything needed (empty, if we find & load stuff, great, but we still need the data struct for the last step)
	beamLookup := dataConvertModels.BeamLocationByPMC{}
	//beamToolVersion := 0
	hkData := dataConvertModels.HousekeepingData{}
	// locSpectraLookup := dataConvertModels.DetectorSampleByPMC{}
	// bulkMaxSpectraLookup := dataConvertModels.DetectorSampleByPMC{}
	contextImgsPerPMC := map[int32]string{}
	pseudoIntensityData := dataConvertModels.PseudoIntensities{}
	pseudoIntensityRanges := []dataConvertModels.PseudoIntensityRange{}
	// rgbuImages := []dataConvertModels.ImageMeta{}
	// discoImages := []dataConvertModels.ImageMeta{}
	// whiteDiscoImage := ""

	elemMapFiles := map[string]string{}
	var opticalBounds image.Rectangle

	for _, file := range files {
		// Check suffix
		if !strings.HasSuffix(file, suffixImageMap) {
			return nil, "", fmt.Errorf("\"%v\" does not have expected suffix: \"%v\"", file, suffixImageMap)
		}

		pos := strings.LastIndex(file, "_")
		if pos < 1 {
			return nil, "", fmt.Errorf("Unexpected file name: \"%v\"", file)
		}

		fileNum := file[pos+1 : len(file)-len(suffixImageMap)]
		if _, err := strconv.Atoi(fileNum); err != nil {
			return nil, "", fmt.Errorf("Unexpected numbering at end of file name: \"%v\"", file)
		} // else do we have a use for num?

		// Work out the element or CP
		if pos < 2 {
			return nil, "", fmt.Errorf("Failed to determine element in file name: \"%v\"", file)
		}

		elem := strings.TrimLeft(file[pos-2:pos], "_ ")
		if elem == "CP" {
			beam, imgBounds, err := im.readOptical(path.Join(importPath, file))
			if err != nil {
				return nil, "", err
			}
			contextImgsPerPMC[0] = file
			beamLookup = beam
			opticalBounds = imgBounds
		} else {
			elemMapFiles[elem] = file
		}
	}

	// Now read the element map files
	for elem, file := range elemMapFiles {
		err := im.readRelativeElementMap(path.Join(importPath, file), opticalBounds, beamLookup, pseudoIntensityData)
		if err != nil {
			return nil, "", err
		}

		//pseudoIdx := int32(len(pseudoIntensityRanges))
		pseudoIntensityRanges = append(pseudoIntensityRanges, dataConvertModels.PseudoIntensityRange{
			Name:  elem,
			Start: 0,
			End:   0,
		})

		//pseudoIntensityData[pseudoIdx] = pseudo
	}

	matchedAlignedImages := []dataConvertModels.MatchedAlignedImageMeta{}
	/*	housekeepingFileNameMeta := gdsfilename.FileNameMeta{}

		data, err := importerutils.MakeFMDatasetOutput(
			beamLookup,
			hkData,
			locSpectraLookup,
			bulkMaxSpectraLookup,
			contextImgsPerPMC,
			pseudoIntensityData,
			pseudoIntensityRanges,
			matchedAlignedImages,
			rgbuImages,
			discoImages,
			whiteDiscoImage,
			housekeepingFileNameMeta,
			datasetIDExpected,
			protos.ScanInstrument_UNKNOWN_INSTRUMENT,
			"A",
			uint32(beamToolVersion),
			log,
		)

		if err != nil {
			return nil, "", err
		}
	*/
	meta := dataConvertModels.FileMetaData{
		/*TargetID: params.TargetID,
		Target:   params.Target,
		SiteID:   params.SiteID,
		Site:     params.Site,
		Title:    params.Title,
		SOL:      params.SOL,*/
	}

	spectraLookup := dataConvertModels.DetectorSampleByPMC{}

	data := &dataConvertModels.OutputData{
		DatasetID:            datasetIDExpected,
		Instrument:           protos.ScanInstrument_UNKNOWN_INSTRUMENT,
		Meta:                 meta,
		DetectorConfig:       "", //params.DetectorConfig,
		BulkQuantFile:        "", //params.BulkQuantFile,
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		MatchedAlignedImages: matchedAlignedImages,
		CreatorUserId:        specialUserIds.PIXLISESystemUserId, // TODO: set a real creator
	}

	if len(contextImgsPerPMC) != 1 {
		return nil, "", fmt.Errorf("Failed to read context image")
	}

	data.SetPMCData(beamLookup, hkData, spectraLookup, contextImgsPerPMC, pseudoIntensityData, map[int32]string{})

	return data, importPath, nil
}

func (im ImageMaps) readOptical(imagePath string) (dataConvertModels.BeamLocationByPMC, image.Rectangle, error) {
	var bounds image.Rectangle
	beams := dataConvertModels.BeamLocationByPMC{}

	// Read the image so we can generate beam locations using find the black pixels as "skipped" locations
	tiffFile, err := os.Open(imagePath)
	if err != nil {
		return beams, bounds, err
	}

	defer tiffFile.Close()

	tiffImg, err := tiff.Decode(tiffFile)
	if err != nil {
		return beams, bounds, err
	}

	bounds = tiffImg.Bounds()
	width := bounds.Dx()

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if y > bounds.Dy()/2 {
				continue
			}

			pixelR, pixelG, pixelB, _ /*pixelA*/ := tiffImg.At(x, y).RGBA()
			if pixelR == 0 &&
				pixelG == 0 &&
				pixelB == 0 { //&&
				//pixelA == 0 {
				// We consider black pixels in the "optical" CP image to be un-scanned points, so don't generate a beam location here
				continue
			}

			loc := dataConvertModels.BeamLocation{
				X:  float32(x),
				Y:  float32(y),
				IJ: map[int32]dataConvertModels.BeamLocationProj{},
			}

			loc.IJ[0] = dataConvertModels.BeamLocationProj{
				I: float32(x),
				J: float32(y),
			}

			pmc := int32(y*width + x)
			beams[pmc] = loc
		}
	}

	// The TIF file should read fine into a PNG as we continue, so stop here
	return beams, bounds, nil
}

func (im ImageMaps) readRelativeElementMap(
	imagePath string,
	opticalBounds image.Rectangle,
	beams dataConvertModels.BeamLocationByPMC,
	pseudoIntensityData dataConvertModels.PseudoIntensities,
) error {
	// Read the image so we can generate beam locations using find the black pixels as "skipped" locations
	tiffFile, err := os.Open(imagePath)
	if err != nil {
		return err
	}

	defer tiffFile.Close()

	tiffImg, err := tiff.Decode(tiffFile)
	if err != nil {
		return err
	}

	bounds := tiffImg.Bounds()

	// Verify the element map is the same resolution as the CP image
	if bounds.Dx() != opticalBounds.Dx() || bounds.Dy() != opticalBounds.Dy() {
		return fmt.Errorf("CP image size (%v x %v) did not match %v element map image size (%v x %v)", opticalBounds.Dx(), opticalBounds.Dy(), imagePath, bounds.Dx(), bounds.Dy())
	}

	width := bounds.Dx()

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			pmc := int32(y*width + x)

			pixelR, pixelG, pixelB, pixelA := tiffImg.At(x, y).RGBA()

			// Check if it's a "valid" location that we have a beam location for
			if _, ok := beams[pmc]; !ok {
				// We don't have beam info for this point, so treat it as a hole
				if pixelR > 0 {
					//fmt.Printf("Warning: Skipping pixel %v,%v value %v because CP image has this marked as a hole\n", x, y, pixelR)
				}
				continue
			}

			if pixelR == 0 &&
				pixelG == 0 &&
				pixelB == 0 &&
				pixelA == 0 {
				// We consider black pixels in the "optical" CP image to be un-scanned points, so don't generate a beam location here
				continue
			}

			loc := dataConvertModels.BeamLocation{
				X:  float32(x),
				Y:  float32(y),
				IJ: map[int32]dataConvertModels.BeamLocationProj{},
			}

			loc.IJ[0] = dataConvertModels.BeamLocationProj{
				I: float32(x),
				J: float32(y),
			}

			arr, ok := pseudoIntensityData[pmc]
			if ok {
				arr = append(arr, float32(pixelR))
			} else {
				arr = []float32{float32(pixelR)}
			}
			pseudoIntensityData[pmc] = arr
		}
	}

	return nil
}
