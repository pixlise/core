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
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const OUTPUT_BASE_DIR = "~/PIXLISE/TESTING"

func main() {
	var inputPath string
	var tileSize int
	var scanID string

	flag.StringVar(&inputPath, "input", "", "Path to input TIFF file (required)")
	flag.IntVar(&tileSize, "tilesize", 254, "Tile size in pixels (default: 254 due to pixel overlap)")
	flag.StringVar(&scanID, "scan", "dummy-scan", "Scan ID (default: dummy-scan)")
	flag.Parse()

	if inputPath == "" {
		printUsage()
		os.Exit(1)
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Input file does not exist: %s\n", inputPath)
		os.Exit(1)
	}

	// Expand home directory
	outputBaseDir := expandHomeDir(OUTPUT_BASE_DIR)

	// Get base image name (without extension)
	baseName := filepath.Base(inputPath)
	ext := filepath.Ext(baseName)
	imageName := baseName[:len(baseName)-len(ext)]

	fmt.Printf("========================================\n")
	fmt.Printf("DeepZoom Tile Generator\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Input:      %s\n", inputPath)
	fmt.Printf("Scan ID:    %s\n", scanID)
	fmt.Printf("Image name: %s\n", imageName)
	fmt.Printf("Tile size:  %d (overlap: 0)\n", tileSize)
	fmt.Printf("Output:     %s/%s/\n", outputBaseDir, scanID)
	fmt.Printf("========================================\n")

	// Use the modular generator function
	opts := DefaultPyramidGenerationOptions()
	opts.TileSize = tileSize

	fmt.Printf("\nGenerating DeepZoom tiles...\n")
	result, err := GeneratePyramidTiles(inputPath, outputBaseDir, imageName, scanID, opts)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}

	// Display metadata
	fmt.Printf("\nTIFF Metadata:\n")
	fmt.Printf("  Pages:          %d\n", result.NumberOfPages)
	fmt.Printf("  Dimensions:     %d x %d (page 0)\n", result.Width, result.Height)
	fmt.Printf("  Bands:          %d\n", result.Bands)
	fmt.Printf("  Interpretation: %v\n", result.Interpretation)

	// Try to get OME XML or image description
	if omeData, err := GetOMEMetadata(inputPath); err == nil && len(omeData) > 0 {
		fmt.Printf("  OME/XML found:  %d bytes\n", len(omeData))
	}

	// Display generation results
	fmt.Printf("\nGenerated tiles for %d page(s):\n", result.NumberOfPages)
	for page := 0; page < result.NumberOfPages; page++ {
		if basePath, ok := result.OutputPaths[page]; ok {
			fmt.Printf("  ✓ Page %d: %s.dzi\n", page, basePath)
		}
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("✓ Done!\n")
	fmt.Printf("========================================\n")
	fmt.Printf("\nGenerated structure:\n")
	fmt.Printf("  %s/\n", outputBaseDir)
	fmt.Printf("    %s/\n", scanID)
	fmt.Printf("      %s/\n", imageName)
	for page := 0; page < result.NumberOfPages; page++ {
		pageName := fmt.Sprintf("page_%d", page)
		fmt.Printf("        %s.dzi\n", pageName)
		fmt.Printf("        %s_files/\n", pageName)
	}
	fmt.Printf("\n")
	fmt.Printf("API endpoint example:\n")
	fmt.Printf("  GET /pyramid-tiles/%s/%s/0/2/3/3\n", scanID, imageName)
	fmt.Printf("  (page 0, level 2, tile x=3, y=3)\n")
	fmt.Printf("\n")
}

func expandHomeDir(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

func printUsage() {
	fmt.Println("ERROR: -input flag is required")
	fmt.Println("")
	fmt.Println("Usage: pyramid-generator -input /path/to/image.tiff [options]")
	fmt.Println("")
	fmt.Println("Generates DeepZoom tiles for all pages in a TIFF file using vips dzsave.")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -input string     Path to input TIFF file (required)")
	fmt.Println("  -tilesize int     Tile size in pixels (default: 254)")
	fmt.Println("  -scan string      Scan ID (default: dummy-scan)")
	fmt.Println("")
	fmt.Println("Output structure:")
	fmt.Println("  ~/PIXLISE/TESTING/<scan-id>/<image-name>_page_0/<image-name>_page_0.dzi")
	fmt.Println("  ~/PIXLISE/TESTING/<scan-id>/<image-name>_page_0/<image-name>_page_0_files/")
	fmt.Println("  ~/PIXLISE/TESTING/<scan-id>/<image-name>_page_1/...")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  pyramid-generator -input z-stack.tif")
	fmt.Println("  pyramid-generator -input z-stack.tif -scan 297796101 -tilesize 512")
	fmt.Println("")
	fmt.Println("Requirements:")
	fmt.Println("  - libvips must be installed (brew install vips or apt install libvips-dev)")
}
