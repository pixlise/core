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

	"google.golang.org/protobuf/proto"
)

// TestImagePyramidProtoSize checks how big the ImagePyramid protobuf is
func TestImagePyramidProtoSize(t *testing.T) {
	testDir := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images")

	testCases := []struct {
		name string
		file string
	}{
		{"Small pyramid (1400x934)", "Generated/test-api-pyramid.tiff"},
		{"Z-stack with pyramids (9728x6144, 9 pages)", "sample_z-stack.tif"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testFile := filepath.Join(testDir, tc.file)
			if _, err := os.Stat(testFile); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", testFile)
			}

			// Get pyramid info
			pyramidProto, err := GetPyramidInfo(testFile)
			if err != nil {
				t.Fatalf("Failed to get pyramid info: %v", err)
			}

			// Serialize the protobuf
			data, err := proto.Marshal(pyramidProto)
			if err != nil {
				t.Fatalf("Failed to marshal proto: %v", err)
			}

			t.Logf("=== %s ===", tc.name)
			t.Logf("File: %s", tc.file)
			t.Logf("Pyramid levels: %d", len(pyramidProto.Pyramid))

			totalTiles := 0
			for i, layer := range pyramidProto.Pyramid {
				totalTiles += len(layer.Tiles)
				t.Logf("  Level %d: %.0fx%.0f, %d tiles",
					i,
					layer.Bounds.Max.X,
					layer.Bounds.Max.Y,
					len(layer.Tiles))
			}

			t.Logf("Total tiles: %d", totalTiles)
			t.Logf("Serialized protobuf size: %d bytes (%.2f KB)",
				len(data), float64(len(data))/1024)

			if len(data) > 1024*1024 {
				t.Logf("⚠️  WARNING: Proto is %.2f MB - might be too large!", float64(len(data))/(1024*1024))
			} else if len(data) > 100*1024 {
				t.Logf("⚠️  Proto is %.2f KB - on the larger side", float64(len(data))/1024)
			} else {
				t.Logf("✓ Proto size is reasonable")
			}
		})
	}
}

// TestImagePyramidProtoStructure shows what's actually in the proto
func TestImagePyramidProtoStructure(t *testing.T) {
	testFile := filepath.Join(os.Getenv("HOME"), "PIXLISE", "Scuffed_Images", "Generated", "test-api-pyramid.tiff")

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skipf("Test file not found: %s", testFile)
	}

	pyramidProto, err := GetPyramidInfo(testFile)
	if err != nil {
		t.Fatalf("Failed to get pyramid info: %v", err)
	}

	t.Logf("=== ImagePyramid Proto Structure ===")
	t.Logf("\nBounds: (%.0f, %.0f) -> (%.0f, %.0f)",
		pyramidProto.Bounds.Min.X, pyramidProto.Bounds.Min.Y,
		pyramidProto.Bounds.Max.X, pyramidProto.Bounds.Max.Y)

	t.Logf("\nLayers: %d", len(pyramidProto.Pyramid))
	for i, layer := range pyramidProto.Pyramid {
		t.Logf("\n  Layer %d:", i)
		t.Logf("    Bounds: (%.0f, %.0f) -> (%.0f, %.0f)",
			layer.Bounds.Min.X, layer.Bounds.Min.Y,
			layer.Bounds.Max.X, layer.Bounds.Max.Y)
		t.Logf("    Tiles: %d", len(layer.Tiles))

		// Show first few tiles
		if len(layer.Tiles) > 0 {
			t.Logf("    First tile bounds: (%.0f, %.0f) -> (%.0f, %.0f)",
				layer.Tiles[0].Bounds.Min.X, layer.Tiles[0].Bounds.Min.Y,
				layer.Tiles[0].Bounds.Max.X, layer.Tiles[0].Bounds.Max.Y)
		}
	}
}
