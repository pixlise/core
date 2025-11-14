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

// ExtractTile extracts a specific tile from a pyramidal TIFF
// zoom: pyramid level (0 = base image, higher = downsampled levels)
// x, y: tile coordinates at that zoom level
// tileSize: tile dimensions (usually 256)
func ExtractTile(pyramidPath string, zoom, x, y, tileSize int) ([]byte, error) {
	// Load the specific pyramid level
	// Level 0 is at page 0, levels 1+ are stored as subIFDs
	var img *vips.Image
	var err error

	if zoom == 0 {
		// Base image at page 0
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Page: 0,
			N:    1,
		})
	} else {
		// Pyramid levels stored as subIFDs
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Subifd: zoom - 1,
			N:      1,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load pyramid level %d: %w", zoom, err)
	}
	defer img.Close()

	// Calculate crop region
	left := x * tileSize
	top := y * tileSize
	width := tileSize
	height := tileSize

	// Get actual image dimensions at this level
	levelWidth := img.Width()
	levelHeight := img.Height()

	// Clamp tile dimensions if at edge
	if left+width > levelWidth {
		width = levelWidth - left
	}
	if top+height > levelHeight {
		height = levelHeight - top
	}

	// Check if tile is completely out of bounds
	if left >= levelWidth || top >= levelHeight || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("tile (%d,%d) out of bounds for level %d", x, y, zoom)
	}

	// Extract the tile region (modifies img in-place)
	err = img.ExtractArea(left, top, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tile area: %w", err)
	}

	// If tile is smaller than expected (edge tile), embed it in a larger canvas
	if width < tileSize || height < tileSize {
		err = img.Embed(0, 0, tileSize, tileSize, &vips.EmbedOptions{
			Extend:     vips.ExtendWhite,
			Background: []float64{255, 255, 255},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to embed tile: %w", err)
		}
	}

	// Encode as JPEG
	buf, err := img.JpegsaveBuffer(&vips.JpegsaveBufferOptions{
		Q:              85,
		OptimizeCoding: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode tile: %w", err)
	}

	return buf, nil
}

// ExtractTilePNG extracts a tile and encodes as PNG (lossless)
func ExtractTilePNG(pyramidPath string, zoom, x, y, tileSize int) ([]byte, error) {
	// Load the specific pyramid level
	// Level 0 is at page 0, levels 1+ are stored as subIFDs
	var img *vips.Image
	var err error

	if zoom == 0 {
		// Base image at page 0
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Page: 0,
			N:    1,
		})
	} else {
		// Pyramid levels stored as subIFDs
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Subifd: zoom - 1,
			N:      1,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load pyramid level %d: %w", zoom, err)
	}
	defer img.Close()

	// Calculate crop region
	left := x * tileSize
	top := y * tileSize
	width := tileSize
	height := tileSize

	// Get actual image dimensions
	levelWidth := img.Width()
	levelHeight := img.Height()

	// Clamp dimensions
	if left+width > levelWidth {
		width = levelWidth - left
	}
	if top+height > levelHeight {
		height = levelHeight - top
	}

	// Check bounds
	if left >= levelWidth || top >= levelHeight || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("tile (%d,%d) out of bounds for level %d", x, y, zoom)
	}

	// Extract tile (modifies img in-place)
	err = img.ExtractArea(left, top, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tile area: %w", err)
	}

	// Embed if needed
	if width < tileSize || height < tileSize {
		err = img.Embed(0, 0, tileSize, tileSize, &vips.EmbedOptions{
			Extend:     vips.ExtendWhite,
			Background: []float64{255, 255, 255},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to embed tile: %w", err)
		}
	}

	// Encode as PNG
	buf, err := img.PngsaveBuffer(&vips.PngsaveBufferOptions{
		Compression: 6,
		Filter:      vips.PngFilterNone,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode tile: %w", err)
	}

	return buf, nil
}

// ExtractPyramidLevel extracts a complete pyramid level and saves it as a JPEG
// level: pyramid level (0 = base image, higher = downsampled levels)
// outputPath: path to save the extracted level image
func ExtractPyramidLevel(pyramidPath string, level int, outputPath string) error {
	// Load the specific pyramid level
	// Level 0 is at page 0, levels 1+ are stored as subIFDs
	var img *vips.Image
	var err error

	if level == 0 {
		// Base image at page 0
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Page: 0,
			N:    1,
		})
	} else {
		// Pyramid levels stored as subIFDs
		img, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Subifd: level - 1,
			N:      1,
		})
	}

	if err != nil {
		return fmt.Errorf("failed to load pyramid level %d: %w", level, err)
	}
	defer img.Close()

	// Save as JPEG
	err = img.Jpegsave(outputPath, &vips.JpegsaveOptions{
		Q:              85,
		OptimizeCoding: true,
	})
	if err != nil {
		return fmt.Errorf("failed to save level %d: %w", level, err)
	}

	return nil
}

// ExtractTileFromPage extracts a tile from a specific page and pyramid level
// Handles both single-page and multi-page pyramids
// page: which page/z-slice (0 for single-page TIFFs)
// level: pyramid level (0 = base image, higher = downsampled levels)
// x, y: tile coordinates at that zoom level
// tileSize: tile dimensions (usually 256)
func ExtractTileFromPage(pyramidPath string, page, level, x, y, tileSize int) ([]byte, error) {
	// Load the specific page and pyramid level
	img, err := GetPageAndLevel(pyramidPath, page, level)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	// Calculate crop region
	left := x * tileSize
	top := y * tileSize
	width := tileSize
	height := tileSize

	// Get actual image dimensions at this level
	levelWidth := img.Width()
	levelHeight := img.Height()

	// Clamp tile dimensions if at edge
	if left+width > levelWidth {
		width = levelWidth - left
	}
	if top+height > levelHeight {
		height = levelHeight - top
	}

	// Check if tile is completely out of bounds
	if left >= levelWidth || top >= levelHeight || width <= 0 || height <= 0 {
		return nil, fmt.Errorf("tile (%d,%d) out of bounds for page %d, level %d", x, y, page, level)
	}

	// Extract the tile region (modifies img in-place)
	err = img.ExtractArea(left, top, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tile area: %w", err)
	}

	// If tile is smaller than expected (edge tile), embed it in a larger canvas
	if width < tileSize || height < tileSize {
		err = img.Embed(0, 0, tileSize, tileSize, &vips.EmbedOptions{
			Extend:     vips.ExtendWhite,
			Background: []float64{255, 255, 255},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to embed tile: %w", err)
		}
	}

	// Encode as JPEG
	buf, err := img.JpegsaveBuffer(&vips.JpegsaveBufferOptions{
		Q:              85,
		OptimizeCoding: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode tile: %w", err)
	}

	return buf, nil
}
