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
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratePyramidalTIFF(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	// Check if test directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skipf("Test directory not found: %s", testDir)
	}

	testFile := filepath.Join(testDir, "sample_5mb.tiff")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Join(testDir, "Generated")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	outputFile := filepath.Join(outputDir, "pyramid.tiff")

	t.Logf("Generating pyramidal TIFF from: %s", testFile)
	t.Logf("Output: %s", outputFile)

	// Generate pyramid
	input := ImageInput{
		Path:    testFile,
		Channel: 0,
	}

	config := GeneratorConfig{
		TileSize:    64,
		Compression: "jpeg",
		Quality:     20,
	}

	err := GeneratePyramidalTIFF(input, outputFile, config)
	if err != nil {
		t.Fatalf("Failed to generate pyramid: %v", err)
	}

	// Verify output file exists
	info, err := os.Stat(outputFile)
	if err != nil {
		t.Fatalf("Output file not created: %v", err)
	}

	t.Logf("✓ Pyramid generated successfully")
	t.Logf("  Size: %.2f MB", float64(info.Size())/(1024*1024))

	// Get pyramid info
	pyramid, err := GetPyramidInfo(outputFile)
	if err != nil {
		t.Fatalf("Failed to get pyramid info: %v", err)
	}

	t.Logf("✓ Pyramid metadata:")
	t.Logf("  Levels: %d", len(pyramid.Pyramid))
	t.Logf("  Bounds: (%.0f, %.0f) to (%.0f, %.0f)",
		pyramid.Bounds.Min.X, pyramid.Bounds.Min.Y,
		pyramid.Bounds.Max.X, pyramid.Bounds.Max.Y)

	for i, layer := range pyramid.Pyramid {
		t.Logf("  Level %d: %.0f×%.0f, %d tiles",
			i,
			layer.Bounds.Max.X,
			layer.Bounds.Max.Y,
			len(layer.Tiles))
	}

	// Extract all pyramid levels for visual inspection
	t.Logf("✓ Extracting pyramid levels for inspection:")
	for i := range pyramid.Pyramid {
		levelFile := filepath.Join(outputDir, fmt.Sprintf("level_%d.jpg", i))
		err = ExtractPyramidLevel(outputFile, i, levelFile)
		if err != nil {
			t.Errorf("Failed to extract level %d: %v", i, err)
			continue
		}
		levelInfo, _ := os.Stat(levelFile)
		t.Logf("  Level %d saved: %s (%.2f KB)",
			i, levelFile, float64(levelInfo.Size())/1024)
	}
}

func TestExtractTile(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	// First generate a pyramid
	testFile := filepath.Join(testDir, "sample_5mb.tiff")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	outputDir := t.TempDir()
	pyramidFile := filepath.Join(outputDir, "pyramid.tiff")

	// Generate
	err := GeneratePyramidalTIFF(
		ImageInput{Path: testFile, Channel: 0},
		pyramidFile,
		GeneratorConfig{TileSize: 256, Compression: "jpeg", Quality: 85},
	)
	if err != nil {
		t.Fatalf("Failed to generate pyramid: %v", err)
	}

	// Extract some tiles
	testCases := []struct {
		zoom int
		x    int
		y    int
	}{
		{0, 0, 0}, // Top-left tile at lowest resolution
		{1, 0, 0}, // Top-left at next level
		{1, 1, 0}, // Second tile horizontally
	}

	for _, tc := range testCases {
		t.Run(t.Name()+"_tile", func(t *testing.T) {
			tileData, err := ExtractTile(pyramidFile, tc.zoom, tc.x, tc.y, 256)
			if err != nil {
				t.Errorf("Failed to extract tile (%d, %d, %d): %v", tc.zoom, tc.x, tc.y, err)
				return
			}

			if len(tileData) == 0 {
				t.Errorf("Tile data is empty")
				return
			}

			t.Logf("✓ Tile (%d, %d, %d) extracted: %d bytes", tc.zoom, tc.x, tc.y, len(tileData))

			// Optionally save tile to verify
			tileFile := filepath.Join(outputDir, t.Name()+".jpg")
			err = os.WriteFile(tileFile, tileData, 0644)
			if err == nil {
				t.Logf("  Saved to: %s", tileFile)
			}
		})
	}
}
