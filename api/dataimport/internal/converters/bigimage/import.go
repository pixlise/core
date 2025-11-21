package importBigImage

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	dataimportModel "github.com/pixlise/core/v4/api/dataimport/models"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// IsBigImageDataSet detects if the import is a big image dataset that needs pyramid tile generation
// Expected directory structure:
//
//	<importPath>/
//	  pyramid/
//	    <image-name>.tif or .tiff    (exactly 1 TIFF file)
//
// The pyramid/ subdirectory should contain exactly one TIFF file which will be processed
// to generate DeepZoom pyramid tiles during import.
func IsBigImageDataSet(importPath string) bool {
	localFS := &fileaccess.FSAccess{}

	files, err := localFS.ListObjects(importPath, "pyramid/")

	if err != nil {
		return false
	}

	// We only care about the pyramid directory, expecting exactly 1 TIFF file in there
	count := 0
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".tif" || ext == ".tiff" {
			count++
		} else {
			// If there's any non-TIFF file, this is not a BigImage dataset
			return false
		}
	}

	return count == 1
}

type BigImage struct {
}

// For importing TIF/TIFF data that will be used to generate DeepZoom pyramids

func (im BigImage) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	localFS := &fileaccess.FSAccess{}

	// Check if we can load the import instructions JSON file
	var params dataimportModel.BreadboardImportParams
	err := localFS.ReadJSON(importPath, "import.json", &params, false)
	if err != nil {
		// If there is no import.json file, we can use some suitable defaults, so just warn here
		//return nil, "", err
		log.Infof("Warning: No import.json found, defaults will be used")
	}

	files, err := localFS.ListObjects(importPath, "pyramid") // Should be a single TIFF file

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

	// Find the single TIFF file in the pyramid directory
	var tiffFile string
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if ext == ".tif" || ext == ".tiff" {
			// Keep the full relative path including "pyramid/" directory
			// file = "pyramid/Multi_page24bpp.tif"
			tiffFile = file
			break
		}
	}

	if tiffFile == "" {
		return nil, "", fmt.Errorf("no TIFF file found in pyramid directory")
	}

	log.Infof("Found pyramid source image: %s", tiffFile)

	// Add the image(s) to PMC(s) with PY_ prefix on the filename part
	// The PY_ prefix signals to output.go:copyImagesToOutput() to generate pyramid tiles
	dir := filepath.Dir(tiffFile)                              // "pyramid"
	base := filepath.Base(tiffFile)                            // "Multi_page24bpp.tif"
	baseWithoutExt := base[:len(base)-len(filepath.Ext(base))] // "Multi_page24bpp"

	// Detect if multi-page TIFF
	fullTiffPath := filepath.Join(importPath, tiffFile)
	pageCount, err := getPageCount(fullTiffPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load tiff (couldn't get page count or invalid dimensions) %s: %v", tiffFile, err)
	}

	log.Infof("TIFF has %d page(s)", pageCount)

	// Create PMC entry for each page
	// All entries point to same source file, but have different destination names
	// sourcePath := filepath.Join(dir, "PY_"+base) // "pyramid/PY_Multi_page24bpp.tif"

	for page := 0; page < pageCount; page++ {
		pmc := int32(page + 1) // PMC starts at 1

		// All pages get _pageN suffix: "pyramid/PY_Multi_page24bpp_page0.tif", etc.
		destName := filepath.Join(dir, fmt.Sprintf("PY_%s_page%d%s", baseWithoutExt, page, filepath.Ext(base)))

		contextImgsPerPMC[pmc] = destName
		log.Infof("Registered page %d for PMC %d: %s", page, pmc, destName)
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
		Title: datasetIDExpected,
	}

	spectraLookup := dataConvertModels.DetectorSampleByPMC{}

	data := &dataConvertModels.OutputData{
		DatasetID:            datasetIDExpected,
		Instrument:           protos.ScanInstrument_UNKNOWN_INSTRUMENT,
		Meta:                 meta,
		DetectorConfig:       "Breadboard", //params.DetectorConfig,
		BulkQuantFile:        "",           //params.BulkQuantFile,
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		MatchedAlignedImages: matchedAlignedImages,
		CreatorUserId:        specialUserIds.PIXLISESystemUserId, // TODO: set a real creator
	}

	if len(contextImgsPerPMC) < 1 {
		return nil, "", fmt.Errorf("Failed to read context image")
	}

	data.SetPMCData(beamLookup, hkData, spectraLookup, contextImgsPerPMC, pseudoIntensityData, map[int32]string{})

	return data, importPath, nil
}

// getPageCount detects how many pages are in a TIFF file without loading the entire file
func getPageCount(tiffPath string) (int, error) {
	// Can also return error if the pages are not same dimensions
	// Load TIFF and read metadata
	// TODO ? : Right now we don't *explicitly* check for dimension limits here, but vips.NewTiffload() will fail if the dimensions are invalid (e.g. too large). TEST this properly.
	img, err := vips.NewTiffload(tiffPath, &vips.TiffloadOptions{
		Page: 0,
		N:    -1, // Loads all pages in a 'toilet paper' strip format.
	})
	if err != nil {
		return 0, fmt.Errorf("failed to load TIFF, potentially incorrect dimensions: %w", err)
	}
	defer img.Close()
	return img.Pages(), err
}
