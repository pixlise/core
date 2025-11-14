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
	"math"

	"github.com/cshum/vipsgen/vips"
	protos "github.com/pixlise/core/v4/generated-protos"
)

type ImageInput struct {
	Path    string // Path to source image
	Channel uint8  // Channel to extract (0 for first/grayscale)
}

type GeneratorConfig struct {
	TileSize    int    // Tile dimensions (default: 256)
	Compression string // "jpeg" or "deflate" (default: "jpeg")
	Quality     int    // JPEG quality 1-100 (default: 85)
}

// GeneratePyramidalTIFF creates a multi-resolution pyramidal TIFF from input images
func GeneratePyramidalTIFF(input ImageInput, outputPath string, config GeneratorConfig) error {
	// Set defaults
	if config.TileSize == 0 {
		config.TileSize = 256
	}
	if config.Compression == "" {
		config.Compression = "jpeg"
	}
	if config.Quality == 0 {
		config.Quality = 85
	}

	// Load source image
	img, err := vips.NewTiffload(input.Path, nil)
	if err != nil {
		return fmt.Errorf("failed to load image %s: %w", input.Path, err)
	}
	defer img.Close()

	// Extract specific channel if multi-band
	if img.Bands() > 1 && input.Channel > 0 {
		if int(input.Channel) >= img.Bands() {
			return fmt.Errorf("channel %d out of range (image has %d bands)", input.Channel, img.Bands())
		}
		err = img.ExtractBand(int(input.Channel), &vips.ExtractBandOptions{N: 1})
		if err != nil {
			return fmt.Errorf("failed to extract channel %d: %w", input.Channel, err)
		}
	}

	// Choose compression
	var compression vips.TiffCompression
	switch config.Compression {
	case "jpeg":
		compression = vips.TiffCompressionJpeg
	case "deflate":
		compression = vips.TiffCompressionDeflate
	default:
		compression = vips.TiffCompressionJpeg
	}

	// Save as pyramidal TIFF
	err = img.Tiffsave(outputPath, &vips.TiffsaveOptions{
		Tile:        true,
		TileWidth:   config.TileSize,
		TileHeight:  config.TileSize,
		Pyramid:     true, // Generate pyramid levels
		Compression: compression,
		Q:           config.Quality, // JPEG quality
		Bigtiff:     true,           // Support >4GB files
		Subifd:      true,           // Makes pyramids into a Subifd instead
	})
	if err != nil {
		return fmt.Errorf("failed to save pyramidal TIFF: %w", err)
	}

	return nil
}

// GetPyramidInfo reads a pyramidal TIFF and returns the ImagePyramid proto
func GetPyramidInfo(pyramidPath string) (*protos.ImagePyramid, error) {
	// Load base image (page 0) to get dimensions
	img, err := vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
		Page: 0,
		N:    1, // Load just first page
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load pyramid: %w", err)
	}

	baseWidth := img.Width()
	baseHeight := img.Height()
	img.Close()

	// Count pyramid levels by trying to load each subIFD until we get an error
	// Level 0 is the base image (page 0), levels 1+ are subIFDs
	levels := 1 // Start with base level
	for subifd := 0; ; subifd++ {
		testImg, err := vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
			Subifd: subifd,
			N:      1,
		})
		if err != nil {
			break
		}
		testImg.Close()
		levels++
	}

	// Build protobuf structure
	pyramid := &protos.ImagePyramid{
		Bounds: &protos.AABB{
			Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
			Max: &protos.Coordinate3D{X: float32(baseWidth), Y: float32(baseHeight), Z: 0},
		},
		Pyramid: make([]*protos.ImagePyramidLayer, 0, levels),
	}

	// Process each pyramid level
	for level := 0; level < levels; level++ {
		var levelImg *vips.Image
		var err error

		if level == 0 {
			// Level 0 is the base image at page 0
			levelImg, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
				Page: 0,
				N:    1,
			})
		} else {
			// Levels 1+ are stored as subIFDs
			levelImg, err = vips.NewTiffload(pyramidPath, &vips.TiffloadOptions{
				Subifd: level - 1,
				N:      1,
			})
		}

		if err != nil {
			return nil, fmt.Errorf("failed to load pyramid level %d: %w", level, err)
		}

		levelWidth := levelImg.Width()
		levelHeight := levelImg.Height()
		levelImg.Close()

		// Calculate tile grid for this level
		tileSize := 256 // TODO: Read from TIFF metadata
		tilesX := int(math.Ceil(float64(levelWidth) / float64(tileSize)))
		tilesY := int(math.Ceil(float64(levelHeight) / float64(tileSize)))

		// Create layer metadata
		layer := &protos.ImagePyramidLayer{
			Bounds: &protos.AABB{
				Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
				Max: &protos.Coordinate3D{X: float32(levelWidth), Y: float32(levelHeight), Z: 0},
			},
			Tiles: make([]*protos.ImageTileSummary, 0, tilesX*tilesY),
		}

		// Generate tile summaries (simplified - no point/polygon data for now)
		for y := 0; y < tilesY; y++ {
			for x := 0; x < tilesX; x++ {
				tileMinX := x * tileSize
				tileMinY := y * tileSize
				tileMaxX := min(tileMinX+tileSize, levelWidth)
				tileMaxY := min(tileMinY+tileSize, levelHeight)

				tileSummary := &protos.ImageTileSummary{
					Bounds: &protos.AABB{
						Min: &protos.Coordinate3D{X: float32(tileMinX), Y: float32(tileMinY), Z: 0},
						Max: &protos.Coordinate3D{X: float32(tileMaxX), Y: float32(tileMaxY), Z: 0},
					},
					Points:   0, // Skip for now
					Polygons: 0, // Skip for now
				}
				layer.Tiles = append(layer.Tiles, tileSummary)
			}
		}

		pyramid.Pyramid = append(pyramid.Pyramid, layer)
	}

	return pyramid, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
