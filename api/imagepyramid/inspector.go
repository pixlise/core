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

package imagepyramid

import (
	"fmt"

	"github.com/cshum/vipsgen/vips"
)

// TIFFInfo describes the structure of a TIFF file
type TIFFInfo struct {
	Width         int  // Width of first page
	Height        int  // Height of first page
	Pages         int  // Number of pages/directories (e.g., z-stack slices)
	PyramidLevels int  // Number of SubIFDs (pyramid levels), 0 if no pyramid
	Bands         int  // Number of color channels
	HasPyramid    bool // True if the TIFF already has pyramid levels
}

// InspectTIFF analyzes a TIFF file and returns its structure
// This tells you whether the file already has pyramids or needs them generated
func InspectTIFF(path string) (*TIFFInfo, error) {
	// Load the TIFF to get metadata
	img, err := vips.NewTiffload(path, &vips.TiffloadOptions{
		Page: 0, // Load first page
		N:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load TIFF: %w", err)
	}
	defer img.Close()

	info := &TIFFInfo{
		Width:  img.Width(),
		Height: img.Height(),
		Bands:  img.Bands(),
	}

	// Get number of pages (for multi-page TIFFs like z-stacks)
	nPages, err := img.GetInt("n-pages")
	if err == nil {
		info.Pages = nPages
	} else {
		info.Pages = 1 // Single page TIFF
	}

	// Get number of SubIFDs (pyramid levels)
	nSubIFDs, err := img.GetInt("n-subifds")
	if err == nil && nSubIFDs > 0 {
		info.PyramidLevels = nSubIFDs
		info.HasPyramid = true
	}

	return info, nil
}

// Print outputs human-readable information about the TIFF
func (info *TIFFInfo) Print() string {
	status := "No pyramid"
	if info.HasPyramid {
		status = fmt.Sprintf("Has %d pyramid levels", info.PyramidLevels)
	}

	return fmt.Sprintf(
		"TIFF: %d x %d, %d page(s), %d band(s) - %s",
		info.Width, info.Height, info.Pages, info.Bands, status,
	)
}

// GetPageAndLevel loads a specific page and pyramid level from a TIFF
// page: which page/directory to load (for z-stacks, this is the z-slice)
// level: which pyramid level to load (0 = full res, 1+ = downsampled)
//
// For TIFFs with pyramids:
//   - level 0: base image at Page
//   - level 1+: SubIFD (level-1)
//
// For TIFFs without pyramids:
//   - only level 0 is available
//
// Returns error if page or level doesn't exist
func GetPageAndLevel(path string, page int, level int) (*vips.Image, error) {
	// Basic validation
	if page < 0 {
		return nil, fmt.Errorf("invalid page %d: page must be >= 0", page)
	}
	if level < 0 {
		return nil, fmt.Errorf("invalid level %d: level must be >= 0", level)
	}

	opts := &vips.TiffloadOptions{
		Page: page,
		N:    1,
	}

	// If requesting a pyramid level > 0, load from SubIFD
	if level > 0 {
		opts.Subifd = level - 1
	}

	img, err := vips.NewTiffload(path, opts)
	if err != nil {
		// vips returns generic error if page/level doesn't exist
		// Make it clearer for API consumers
		return nil, fmt.Errorf("page %d or level %d does not exist in TIFF: %w", page, level, err)
	}

	return img, nil
}
