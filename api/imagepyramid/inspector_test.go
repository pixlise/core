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
	"os"
	"path/filepath"
	"testing"
)

func TestInspectTIFF(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	testCases := []struct {
		name           string
		file           string
		expectPyramid  bool
		expectPages    int
	}{
		{
			name:          "Z-stack with pyramids",
			file:          "sample_z-stack.tif",
			expectPyramid: true,
			expectPages:   9,
		},
		{
			name:          "Simple TIFF without pyramid",
			file:          "sample_5mb.tiff",
			expectPyramid: false,
			expectPages:   1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(testDir, tc.file)
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", testFile)
			}

			// Inspect the file
			info, err := InspectTIFF(testFile)
			if err != nil {
				t.Fatalf("Failed to inspect: %v", err)
			}

			t.Logf("%s", info.Print())

			// Verify expectations
			if info.HasPyramid != tc.expectPyramid {
				t.Errorf("Expected HasPyramid=%v, got %v", tc.expectPyramid, info.HasPyramid)
			}

			if info.Pages != tc.expectPages {
				t.Errorf("Expected %d pages, got %d", tc.expectPages, info.Pages)
			}

			if info.HasPyramid {
				t.Logf("  ✓ File already has %d pyramid levels", info.PyramidLevels)
				t.Logf("  → Can serve tiles directly from existing pyramid")
			} else {
				t.Logf("  ✗ File has no pyramid")
				t.Logf("  → Need to generate pyramid first")
			}
		})
	}
}

func TestGetPageAndLevel(t *testing.T) {
	testFile := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images", "sample_z-stack.tif")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	// Inspect first
	info, err := InspectTIFF(testFile)
	if err != nil {
		t.Fatalf("Failed to inspect: %v", err)
	}

	t.Logf("Testing access to %s", info.Print())

	// Test accessing different pages and levels
	testCases := []struct {
		page  int
		level int
	}{
		{0, 0}, // Page 0, full resolution
		{0, 1}, // Page 0, pyramid level 1
		{0, 3}, // Page 0, pyramid level 3
		{5, 0}, // Page 5 (different z-slice), full resolution
		{5, 2}, // Page 5, pyramid level 2
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			img, err := GetPageAndLevel(testFile, tc.page, tc.level)
			if err != nil {
				t.Errorf("Failed to load page %d, level %d: %v", tc.page, tc.level, err)
				return
			}
			defer img.Close()

			t.Logf("Page %d, Level %d: %d×%d",
				tc.page, tc.level, img.Width(), img.Height())
		})
	}
}

func TestWorkflowDecision(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	t.Log("=== Demonstrating workflow decision logic ===\n")

	files := []string{"sample_z-stack.tif", "sample_5mb.tiff"}

	for _, filename := range files {
		testFile := filepath.Join(testDir, filename)
		if _, err := os.Stat(testFile); os.IsNotExist(err) {
			continue
		}

		info, err := InspectTIFF(testFile)
		if err != nil {
			t.Logf("Error inspecting %s: %v", filename, err)
			continue
		}

		t.Logf("File: %s", filename)
		t.Logf("  %s", info.Print())

		// Decide what to do based on inspection
		if info.HasPyramid {
			t.Logf("  Decision: ✓ Use existing pyramid")
			t.Logf("    - Serve tiles directly from %d pyramid levels", info.PyramidLevels)
			if info.Pages > 1 {
				t.Logf("    - Handle %d pages (z-stack)", info.Pages)
			}
		} else {
			t.Logf("  Decision: ✗ Generate pyramid first")
			t.Logf("    - Call GeneratePyramidalTIFF()")
			t.Logf("    - Then serve tiles from generated pyramid")
		}
		t.Logf("")
	}
}
