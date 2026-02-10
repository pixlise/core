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
	"os"
	"path/filepath"
	"testing"

	"github.com/cshum/vipsgen/vips816"
)

func TestTiffLoading(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	// Check if test directory exists
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skipf("Test directory not found: %s", testDir)
	}

	testFiles := []struct {
		name     string
		filename string
	}{
		{"Multi-page 24bpp", "Multi_page24bpp.tif"},
		{"Multi-page example", "multipage_tiff_example.tif"},
		{"5MB sample", "sample_5mb.tiff"},
		{"Z-stack", "sample_z-stack.tif"},
	}

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(testDir, tc.filename)

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", filePath)
			}

			// Get file size
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("Cannot stat file: %v", err)
			}
			t.Logf("File size: %.2f MB", float64(fileInfo.Size())/(1024*1024))

			// Load the TIFF (page 0)
			img, err := vips.NewTiffload(filePath, &vips.TiffloadOptions{
				Page: 0,
				N:    1,
			})
			if err != nil {
				t.Fatalf("Failed to load TIFF: %v", err)
			}
			defer img.Close()

			// Verify image properties
			width := img.Width()
			height := img.Height()
			bands := img.Bands()
			pages := img.Pages()

			t.Logf("✓ Loaded successfully")
			t.Logf("  Dimensions: %dx%d", width, height)
			t.Logf("  Bands: %d", bands)
			t.Logf("  Format: %v", img.Format())
			t.Logf("  Interpretation: %v (%s)", img.Interpretation(), interpretationName(img.Interpretation()))
			if pages > 1 {
				t.Logf("  Total pages: %d", pages)
			}

			// Basic sanity checks
			if width <= 0 || height <= 0 {
				t.Errorf("Invalid dimensions: %dx%d", width, height)
			}
			if bands <= 0 {
				t.Errorf("Invalid band count: %d", bands)
			}

			// Try to get a pixel value
			if width > 0 && height > 0 {
				centerX := width / 2
				centerY := height / 2
				pixel, err := img.Getpoint(centerX, centerY, nil)
				if err != nil {
					t.Errorf("Failed to get pixel: %v", err)
				} else {
					t.Logf("  Center pixel (%d,%d): %v", centerX, centerY, pixel)
				}
			}
		})
	}
}

func TestMultiPageExtraction(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")
	filePath := filepath.Join(testDir, "multipage_tiff_example.tif")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", filePath)
	}

	// Load first to check page count
	img, err := vips.NewTiffload(filePath, &vips.TiffloadOptions{
		Page: 0,
		N:    1,
	})
	if err != nil {
		t.Fatalf("Failed to load TIFF: %v", err)
	}
	pages := img.Pages()
	img.Close()

	t.Logf("Total pages in file: %d", pages)

	if pages <= 1 {
		t.Skip("Not a multi-page TIFF")
	}

	// Try loading different pages
	testPages := []int{0, pages / 2, pages - 1}
	for _, pageNum := range testPages {
		t.Run(t.Name()+"_page_"+string(rune(pageNum+'0')), func(t *testing.T) {
			pageImg, err := vips.NewTiffload(filePath, &vips.TiffloadOptions{
				Page: pageNum,
				N:    1,
			})
			if err != nil {
				t.Errorf("Failed to load page %d: %v", pageNum, err)
				return
			}
			defer pageImg.Close()

			t.Logf("✓ Page %d loaded: %dx%d", pageNum, pageImg.Width(), pageImg.Height())
		})
	}
}

func TestTiffToPngConversion(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")
	filePath := filepath.Join(testDir, "sample_5mb.tiff")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", filePath)
	}

	// Load TIFF
	img, err := vips.NewTiffload(filePath, &vips.TiffloadOptions{
		Page: 0,
		N:    1,
	})
	if err != nil {
		t.Fatalf("Failed to load TIFF: %v", err)
	}
	defer img.Close()

	// Save as PNG to temp file
	tmpFile := filepath.Join(t.TempDir(), "output.png")
	err = img.Pngsave(tmpFile, nil)
	if err != nil {
		t.Fatalf("Failed to save PNG: %v", err)
	}

	// Verify PNG was created
	pngInfo, err := os.Stat(tmpFile)
	if err != nil {
		t.Fatalf("PNG file not created: %v", err)
	}

	t.Logf("✓ Converted TIFF to PNG")
	t.Logf("  PNG size: %.2f MB", float64(pngInfo.Size())/(1024*1024))
	t.Logf("  Location: %s", tmpFile)
}
