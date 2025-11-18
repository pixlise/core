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
	"strings"

	"github.com/cshum/vipsgen/vips"
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

	// Step 1: Load TIFF and read metadata
	fmt.Printf("\nReading TIFF metadata...\n")
	img, err := vips.NewTiffload(inputPath, &vips.TiffloadOptions{
		Page: 0,
		N:    1,
	})
	if err != nil {
		fmt.Printf("ERROR: Failed to load TIFF: %v\n", err)
		os.Exit(1)
	}
	defer img.Close()

	// Get metadata
	nPages := 1
	if pagesVal, err := img.GetInt("n-pages"); err == nil {
		nPages = pagesVal
	}

	bands := img.Bands()
	width := img.Width()
	height := img.Height()
	interpretation := img.Interpretation()

	fmt.Printf("  Pages:          %d\n", nPages)
	fmt.Printf("  Dimensions:     %d x %d (page 0)\n", width, height)
	fmt.Printf("  Bands:          %d\n", bands)
	fmt.Printf("  Interpretation: %v\n", interpretation)

	// Try to get OME XML or image description
	if desc, err := img.GetString("image-description"); err == nil && len(desc) > 0 {
		if strings.Contains(desc, "OME") || strings.Contains(desc, "<?xml") {
			fmt.Printf("  OME/XML found:  %d bytes\n", len(desc))
		}
	}

	// Step 2: Process each page with dzsave
	fmt.Printf("\nGenerating DeepZoom tiles for %d page(s)...\n", nPages)

	for page := 0; page < nPages; page++ {
		fmt.Printf("\n  Page %d/%d:\n", page+1, nPages)

		// Construct output paths
		// Structure: scan/imageName/page_N/
		pageName := fmt.Sprintf("page_%d", page)
		outputDir := filepath.Join(outputBaseDir, scanID, imageName, pageName)

		// Create output directory
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("    ERROR: Failed to create output directory: %v\n", err)
			continue
		}

		// Output base (dzsave will append .dzi and _files/)
		outputBase := filepath.Join(outputDir, pageName)

		fmt.Printf("    Generating tiles...\n")

		// Load this specific page using vipsgen
		pageImg, err := vips.NewTiffload(inputPath, &vips.TiffloadOptions{
			Page: page,
			N:    1,
		})
		if err != nil {
			fmt.Printf("    ERROR: Failed to load page %d: %v\n", page, err)
			continue
		}

		// Use vipsgen Dzsave
		err = pageImg.Dzsave(outputBase, &vips.DzsaveOptions{
			Imagename: pageName,
			Suffix:    ".jpg",
			Q:         85,
			Depth:     vips.DzDepthOnetile,
			Overlap:   0,
			TileSize:  tileSize,
		})

		pageImg.Close()

		if err != nil {
			fmt.Printf("    ERROR: dzsave failed: %v\n", err)
			continue
		}

		fmt.Printf("    ✓ Tiles generated\n")
		fmt.Printf("       Metadata: %s.dzi\n", outputBase)
		fmt.Printf("       Tiles:    %s_files/\n", outputBase)
	}

	fmt.Printf("\n========================================\n")
	fmt.Printf("✓ Done!\n")
	fmt.Printf("========================================\n")
	fmt.Printf("\nGenerated structure:\n")
	fmt.Printf("  %s/\n", outputBaseDir)
	fmt.Printf("    %s/\n", scanID)
	for page := 0; page < nPages; page++ {
		pageImageName := fmt.Sprintf("%s_page_%d", imageName, page)
		fmt.Printf("      %s/\n", pageImageName)
		fmt.Printf("        %s.dzi\n", pageImageName)
		fmt.Printf("        %s_files/\n", pageImageName)
	}
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
