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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cshum/vipsgen/vips"
)

// PyramidGenerationOptions contains configuration for pyramid tile generation
type PyramidGenerationOptions struct {
	// TileSize is the size of each tile in pixels (default: 254)
	TileSize int

	// JPEGQuality is the JPEG compression quality 0-100 (default: 85)
	JPEGQuality int

	// Overlap is the pixel overlap between tiles (default: 0)
	Overlap int
}

// PyramidGenerationResult contains information about the generated pyramid
type PyramidGenerationResult struct {
	// NumberOfPages is the number of pages in the source TIFF
	NumberOfPages int

	// Width is the width of the first page in pixels
	Width int

	// Height is the height of the first page in pixels
	Height int

	// Bands is the number of color bands (e.g., 3 for RGB)
	Bands int

	// Interpretation describes the color interpretation
	Interpretation vips.Interpretation

	// OutputPaths contains the base paths for each generated page
	// Format: map[pageNumber]basePath where basePath is the .dzi file without extension
	OutputPaths map[int]string
}

// DefaultPyramidGenerationOptions returns the default options for pyramid generation
func DefaultPyramidGenerationOptions() PyramidGenerationOptions {
	return PyramidGenerationOptions{
		TileSize:    254,
		JPEGQuality: 85,
		Overlap:     -1,
	}
}

// GeneratePyramidTiles generates DeepZoom tiles for all pages in a TIFF file
//
// Parameters:
//   - inputTiffPath: Path to the input TIFF file
//   - outputBaseDir: Base directory where tiles will be saved
//   - imageName: Name to use for the image (will be used in directory structure)
//   - scanID: Scan ID (will be used in directory structure)
//   - opts: Generation options (use DefaultPyramidGenerationOptions() for defaults)
//
// Output structure:
//
//	outputBaseDir/
//	  scanID/
//	    imageName/
//	      page_0.dzi
//	      page_0_files/
//	      page_1.dzi
//	      page_1_files/
//	      ...
//
// Returns:
//   - PyramidGenerationResult containing metadata about the generated tiles
//   - error if generation fails
func GeneratePyramidTiles(
	inputTiffPath string,
	outputBaseDir string,
	imageName string,
	scanID string,
	opts PyramidGenerationOptions,
) (*PyramidGenerationResult, error) {

	// Step 1: Load TIFF and read metadata from first page
	img, err := vips.NewTiffload(inputTiffPath, &vips.TiffloadOptions{
		Page: 0,
		N:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load TIFF: %w", err)
	}
	defer img.Close()

	// Get metadata
	nPages := 1
	if pagesVal, err := img.GetInt("n-pages"); err == nil {
		nPages = pagesVal
	}

	result := &PyramidGenerationResult{
		NumberOfPages:  nPages,
		Width:          img.Width(),
		Height:         img.Height(),
		Bands:          img.Bands(),
		Interpretation: img.Interpretation(),
		OutputPaths:    make(map[int]string),
	}

	// Step 2: Process each page with dzsave
	// Create output directory (all pages go under imageName directory)
	outputDir := filepath.Join(outputBaseDir, scanID, imageName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	for page := 0; page < nPages; page++ {
		// Construct output paths
		// Structure: outputBaseDir/scanID/imageName/page_N.dzi and page_N_files/
		pageName := fmt.Sprintf("page_%d", page)

		// Output base (dzsave will append .dzi and _files/)
		outputBase := filepath.Join(outputDir, pageName)

		// Load this specific page using vipsgen
		pageImg, err := vips.NewTiffload(inputTiffPath, &vips.TiffloadOptions{
			Page: page,
			N:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load page %d: %w", page, err)
		}

		// Use vipsgen Dzsave with the provided options
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
			Q:         opts.JPEGQuality,
			Depth:     vips.DzDepthOnetile,
			Overlap:   opts.Overlap,
			TileSize:  opts.TileSize,
		})

		pageImg.Close()

		if err != nil {
			return nil, fmt.Errorf("dzsave failed for page %d: %w", page, err)
		}

		// Store the output path for this page
		result.OutputPaths[page] = outputBase
	}

	return result, nil
}

// GetImageNameFromPath extracts the image name (without extension) from a file path
func GetImageNameFromPath(path string) string {
	baseName := filepath.Base(path)
	ext := filepath.Ext(baseName)
	return baseName[:len(baseName)-len(ext)]
}

// GetOMEMetadata attempts to extract OME/XML metadata from a TIFF file
func GetOMEMetadata(inputTiffPath string) (string, error) {
	img, err := vips.NewTiffload(inputTiffPath, &vips.TiffloadOptions{
		Page: 0,
		N:    1,
	})
	if err != nil {
		return "", fmt.Errorf("failed to load TIFF: %w", err)
	}
	defer img.Close()

	if desc, err := img.GetString("image-description"); err == nil && len(desc) > 0 {
		if strings.Contains(desc, "OME") || strings.Contains(desc, "<?xml") {
			return desc, nil
		}
	}

	return "", nil
}
