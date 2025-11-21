// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package pyramid provides functionality for generating DeepZoom pyramid tiles
// from TIFF files during the dataset import process.
package pyramid

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/cshum/vipsgen/vips"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// DZI XML structures for parsing .dzi metadata files
type dziImage struct {
	XMLName  xml.Name `xml:"Image"`
	Format   string   `xml:"Format,attr"`
	Overlap  int      `xml:"Overlap,attr"`
	TileSize int      `xml:"TileSize,attr"`
	Size     dziSize  `xml:"Size"`
}

type dziSize struct {
	Width  int `xml:"Width,attr"`
	Height int `xml:"Height,attr"`
}

type pyramidResult struct {
	NumberOfPages int
	Width         int
	Height        int
}

// ImportBigTIFF generates DeepZoom pyramid tiles from a TIFF file and returns metadata
// for storage in MongoDB.
//
// This function is designed for use in the dataset import pipeline.
// It generates tiles for ALL pages but only returns metadata for page 0,
// assuming all pages have identical dimensions and structure.
//
// Parameters:
//   - fromImgFile: Path to source TIFF (e.g., "/path/pyramid/Multi_page24bpp.tif")
//   - outImgFile: Base output path (e.g., "/path/output-Images/BigTiff/PY_Multi_page24bpp.png")
//     Note: The .png extension is ignored; we create a directory structure instead
//   - jobLog: Logger for import pipeline
//
// Returns:
//   - *protos.ImagePyramid: Metadata about the pyramid structure (based on page 0 only)
//   - error: If generation or parsing fails
//
// Output structure created:
//
//	outputDir/imageName/
//	  page_0.dzi
//	  page_0_files/0/, page_0_files/1/, ...
//	  page_1.dzi
//	  page_1_files/0/, page_1_files/1/, ...
//	  ... (all pages generated)
//
// NOTE: Only page 0 metadata is returned in ImagePyramid proto.
// All pages are assumed to have identical dimensions and tile structure.
func ImportBigTIFF(fromImgFile string, outImgFile string, pageNum int, jobLog logger.ILogger) (*protos.ImagePyramid, error) {
	// Parse parameters from paths
	// fromImgFile = /path/pyramid/PY_Multi_page24bpp.tif (has PY_ prefix, but actual file doesn't)
	// outImgFile  = /path/output-Images/BigTiff/PY_Multi_page24bpp.png (extension ignored)

	// Strip PY_ prefix from fromImgFile to get actual TIFF path
	// The PY_ prefix is used as a signal in the pipeline, but the actual file doesn't have it
	actualTiffPath := fromImgFile
	if strings.HasPrefix(filepath.Base(fromImgFile), "PY_") {
		dir := filepath.Dir(fromImgFile)
		base := filepath.Base(fromImgFile)
		actualTiffPath = filepath.Join(dir, strings.TrimPrefix(base, "PY_"))
	}

	// Extract clean image name (remove PY_ prefix and extension)
	baseName := filepath.Base(outImgFile)                             // "PY_Multi_page24bpp.png" or "PY_Multi_page24bpp_page1.png"
	baseName = strings.TrimPrefix(baseName, "PY_")                    // "Multi_page24bpp.png" or "Multi_page24bpp_page1.png"
	imageName := baseName[:len(baseName)-len(filepath.Ext(baseName))] // "Multi_page24bpp" or "Multi_page24bpp_page1"

	// Get output directory and scan ID
	outputDir := filepath.Dir(outImgFile) // "/path/output-Images/BigTiff"
	scanID := filepath.Base(outputDir)    // "BigTiff"

	jobLog.Infof("Generating DeepZoom pyramid tiles for page %d: %s", pageNum, imageName)
	jobLog.Infof("  Source: %s (actual: %s)", fromImgFile, actualTiffPath)
	jobLog.Infof("  Output: %s/%s/", outputDir, imageName)

	// Generate tiles for ONLY the specified page
	result, err := generatePyramidTiles(actualTiffPath, outputDir, imageName, scanID, pageNum, jobLog)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pyramid tiles: %w", err)
	}

	jobLog.Infof("Generated page %d with dimensions %dx%d", pageNum, result.Width, result.Height)

	// Parse pyramid.dzi to extract metadata
	page0DziPath := filepath.Join(outputDir, imageName, "pyramid.dzi")
	dzi, err := parseDZIFile(page0DziPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page_0.dzi: %w", err)
	}

	jobLog.Infof("Parsed DZI metadata: %dx%d, tileSize=%d, overlap=%d",
		dzi.Size.Width, dzi.Size.Height, dzi.TileSize, dzi.Overlap)

	// Build ImagePyramid proto from page 0 metadata
	pyramid := buildImagePyramidProto(dzi, scanID, imageName, jobLog)

	jobLog.Infof("Created ImagePyramid proto with %d layers (zoom levels)", len(pyramid.Pyramid))

	return pyramid, nil
}

// generatePyramidTiles is the core tile generation logic (copied from pyramid-generator)
func generatePyramidTiles(inputTiffPath string, outputBaseDir string, imageName string, scanID string, pageNum int, jobLog logger.ILogger) (*pyramidResult, error) {
	// Load ONLY the specified page to get dimensions
	img, err := vips.NewTiffload(inputTiffPath, &vips.TiffloadOptions{
		Page: pageNum,
		N:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load TIFF page %d: %w", pageNum, err)
	}
	defer img.Close()

	result := &pyramidResult{
		NumberOfPages: 1, // We only process one page now
		Width:         img.Width(),
		Height:        img.Height(),
	}

	// Create output directory
	// Note: outputBaseDir already includes the scanID, so just append imageName
	outputDir := filepath.Join(outputBaseDir, imageName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Process ONLY the specified page with dzsave
	// Structure: outputBaseDir/imageName/pyramid.dzi and pyramid_files/
	pageName := "pyramid"
	outputBase := filepath.Join(outputDir, pageName)

	// Load this specific page
	pageImg, err := vips.NewTiffload(inputTiffPath, &vips.TiffloadOptions{
		Page: pageNum,
		N:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load page %d: %w", pageNum, err)
	}

	// Use vipsgen Dzsave
	// IMPORTANT: These options are carefully chosen:
	// - Imagename: The base name for generated files
	// - Suffix: .jpg for JPEG tiles
	// - Q: JPEG quality (85 = good balance of quality/size)
	// - Depth: DzDepthOnetile means generate tiles at all zoom levels
	// - Overlap: 0 means no pixel overlap between tiles (can be changed if needed)
	// - TileSize: Size of each tile (254 is default, accounts for overlap)
	err = pageImg.Dzsave(outputBase, &vips.DzsaveOptions{
		Imagename: pageName,
		Suffix:    ".jpg",
		Q:         85,
		Depth:     vips.DzDepthOnetile,
		Overlap:   0,
		TileSize:  254,
	})

	pageImg.Close()

	if err != nil {
		return nil, fmt.Errorf("dzsave failed for page %d: %w", pageNum, err)
	}

	return result, nil
}

// parseDZIFile reads and parses a .dzi XML file
func parseDZIFile(path string) (*dziImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read DZI file: %w", err)
	}

	var dzi dziImage
	if err := xml.Unmarshal(data, &dzi); err != nil {
		return nil, fmt.Errorf("failed to parse DZI XML: %w", err)
	}

	return &dzi, nil
}

// buildImagePyramidProto constructs an ImagePyramid proto from DZI metadata
// NOTE: Only uses page 0 metadata, assumes all pages identical
func buildImagePyramidProto(dzi *dziImage, scanID string, imageName string, jobLog logger.ILogger) *protos.ImagePyramid {
	// Calculate number of zoom levels
	maxDim := float64(dzi.Size.Width)
	if dzi.Size.Height > dzi.Size.Width {
		maxDim = float64(dzi.Size.Height)
	}
	numLevels := int(math.Ceil(math.Log2(maxDim/float64(dzi.TileSize)))) + 1

	jobLog.Infof("Calculated %d zoom levels for image dimensions %dx%d",
		numLevels, dzi.Size.Width, dzi.Size.Height)

	// Overall bounds (using page 0 dimensions)
	bounds := &protos.AABB{
		Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
		Max: &protos.Coordinate3D{X: float32(dzi.Size.Width), Y: float32(dzi.Size.Height), Z: 0},
	}

	// Build layers (one per zoom level)
	layers := make([]*protos.ImagePyramidLayer, numLevels)
	for level := 0; level < numLevels; level++ {
		// Calculate dimensions at this level
		scale := math.Pow(2, float64(numLevels-level-1))
		levelWidth := int(math.Ceil(float64(dzi.Size.Width) / scale))
		levelHeight := int(math.Ceil(float64(dzi.Size.Height) / scale))

		// Calculate number of tiles at this level
		tilesX := int(math.Ceil(float64(levelWidth) / float64(dzi.TileSize)))
		tilesY := int(math.Ceil(float64(levelHeight) / float64(dzi.TileSize)))

		// Create tile summaries (with points=0, polygons=0 for now)
		tiles := make([]*protos.ImageTileSummary, 0, tilesX*tilesY)
		for y := 0; y < tilesY; y++ {
			for x := 0; x < tilesX; x++ {
				// Calculate tile bounds
				tileX := float32(x * dzi.TileSize)
				tileY := float32(y * dzi.TileSize)
				tileW := float32(dzi.TileSize)
				tileH := float32(dzi.TileSize)

				// Clamp to level dimensions
				if tileX+tileW > float32(levelWidth) {
					tileW = float32(levelWidth) - tileX
				}
				if tileY+tileH > float32(levelHeight) {
					tileH = float32(levelHeight) - tileY
				}

				tiles = append(tiles, &protos.ImageTileSummary{
					Bounds: &protos.AABB{
						Min: &protos.Coordinate3D{X: tileX, Y: tileY, Z: 0},
						Max: &protos.Coordinate3D{X: tileX + tileW, Y: tileY + tileH, Z: 0},
					},
					Points:   0, // TODO: Will be populated when overlay data added
					Polygons: 0, // TODO: Will be populated when overlay data added
				})
			}
		}

		layers[level] = &protos.ImagePyramidLayer{
			Bounds: &protos.AABB{
				Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
				Max: &protos.Coordinate3D{X: float32(levelWidth), Y: float32(levelHeight), Z: 0},
			},
			Tiles: tiles,
		}
	}

	// Image prefix (base path for tile files)
	// Points to the image directory containing all pages
	// Tiles are at: {imagePrefix}/page_{N}_files/{level}/{x}_{y}.jpg
	imagePrefix := filepath.Join(scanID, imageName)

	return &protos.ImagePyramid{
		Bounds:        bounds,
		Pyramid:       layers,
		ImagePrefixes: []string{imagePrefix},
	}
}
