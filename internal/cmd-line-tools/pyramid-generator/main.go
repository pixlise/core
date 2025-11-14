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
	"time"

	"github.com/pixlise/core/v4/api/imagepyramid"
)

func main() {
	var inputPath string
	var outputPath string
	var scanID string
	var tileSize int
	var quality int
	var compression string
	var inspect bool

	flag.StringVar(&inputPath, "input", "", "Path to input TIFF file (required)")
	flag.StringVar(&outputPath, "output", "", "Path to output pyramid TIFF (default: ~/PIXLISE/Pyramids/{scanId}/{filename}/pyramid.tiff)")
	flag.StringVar(&scanID, "scan", "Testing", "Scan ID for organizing output (default: Testing)")
	flag.IntVar(&tileSize, "tile-size", 256, "Tile size in pixels (default: 256)")
	flag.IntVar(&quality, "quality", 85, "JPEG quality 1-100 (default: 85)")
	flag.StringVar(&compression, "compression", "jpeg", "Compression: jpeg or deflate (default: jpeg)")
	flag.BoolVar(&inspect, "inspect", false, "Inspect existing pyramid and show metadata")
	flag.Parse()

	if inputPath == "" {
		fmt.Println("Error: -input flag is required")
		fmt.Println("")
		fmt.Println("Usage: pyramid-generator -input /path/to/image.tiff [options]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -input string       Path to input TIFF file (required)")
		fmt.Println("  -output string      Output path (default: ~/PIXLISE/Pyramids/{scan}/{filename}/pyramid.tiff)")
		fmt.Println("  -scan string        Scan ID for organizing output (default: Testing)")
		fmt.Println("  -tile-size int      Tile size in pixels (default: 256)")
		fmt.Println("  -quality int        JPEG quality 1-100 (default: 85)")
		fmt.Println("  -compression string Compression: jpeg or deflate (default: jpeg)")
		fmt.Println("  -inspect            Inspect existing pyramid and show metadata")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  # Generate pyramid for testing")
		fmt.Println("  pyramid-generator -input /data/my_image.tif")
		fmt.Println("")
		fmt.Println("  # Generate with custom quality")
		fmt.Println("  pyramid-generator -input image.tif -quality 90")
		fmt.Println("")
		fmt.Println("  # Inspect existing pyramid")
		fmt.Println("  pyramid-generator -input pyramid.tiff -inspect")
		os.Exit(1)
	}

	// Check if input exists
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Printf("ERROR: Input file does not exist: %s\n", inputPath)
		os.Exit(1)
	}

	// If inspecting, show pyramid info and exit
	if inspect {
		inspectPyramid(inputPath)
		return
	}

	// Determine output path
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("ERROR: Cannot get home directory: %v\n", err)
			os.Exit(1)
		}

		// Extract filename without extension
		filename := filepath.Base(inputPath)
		ext := filepath.Ext(filename)
		nameWithoutExt := filename[:len(filename)-len(ext)]

		outputPath = filepath.Join(homeDir, "PIXLISE", "Pyramids", scanID, nameWithoutExt, "pyramid.tiff")

		// Create output directory
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("ERROR: Cannot create output directory: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("========================================\n")
	fmt.Printf("Pyramidal TIFF Generator\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Input:       %s\n", inputPath)
	fmt.Printf("Output:      %s\n", outputPath)
	fmt.Printf("Scan ID:     %s\n", scanID)
	fmt.Printf("Tile size:   %d x %d\n", tileSize, tileSize)
	fmt.Printf("Quality:     %d\n", quality)
	fmt.Printf("Compression: %s\n", compression)
	fmt.Printf("========================================\n")

	// Get input file size
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot stat input file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Input size:  %.2f MB\n", float64(inputInfo.Size())/(1024*1024))
	fmt.Printf("========================================\n")

	// First inspect the input to see if it already has pyramids
	fmt.Printf("Inspecting input TIFF...\n")
	info, err := imagepyramid.InspectTIFF(inputPath)
	if err != nil {
		fmt.Printf("WARNING: Could not inspect TIFF: %v\n", err)
	} else {
		fmt.Printf("  %s\n", info.Print())

		if info.HasPyramid {
			fmt.Printf("\nℹ️  Input already has pyramids! You can use it directly.\n")
			fmt.Printf("   Consider copying to output location instead of regenerating.\n")
		}
		fmt.Printf("========================================\n")
	}

	// Generate pyramid
	fmt.Printf("Generating pyramid... (this may take a while)\n")
	startTime := time.Now()

	config := imagepyramid.GeneratorConfig{
		TileSize:    tileSize,
		Compression: compression,
		Quality:     quality,
	}

	input := imagepyramid.ImageInput{
		Path:    inputPath,
		Channel: 0, // First channel
	}

	err = imagepyramid.GeneratePyramidalTIFF(input, outputPath, config)
	if err != nil {
		fmt.Printf("ERROR: Pyramid generation failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)

	// Get output file size
	outputInfo, err := os.Stat(outputPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot stat output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("========================================\n")
	fmt.Printf("✅ SUCCESS!\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Generation time: %v\n", duration)
	fmt.Printf("Output size:     %.2f MB\n", float64(outputInfo.Size())/(1024*1024))
	fmt.Printf("Compression:     %.1f%% of original\n", float64(outputInfo.Size())/float64(inputInfo.Size())*100)
	fmt.Printf("========================================\n")

	// Inspect the generated pyramid
	fmt.Printf("\nInspecting generated pyramid:\n")
	inspectPyramid(outputPath)

	fmt.Printf("\n✅ Pyramid ready for testing!\n")
	fmt.Printf("\nTo test the API endpoints:\n")
	fmt.Printf("  1. Start the API: go run ./internal/api\n")
	fmt.Printf("  2. Get metadata: curl http://localhost:8080/pyramid-info/%s/%s\n", scanID, filepath.Base(inputPath))
	fmt.Printf("  3. Get a tile:   curl http://localhost:8080/pyramid-tiles/%s/%s/0/0/0/0 > tile.jpg\n", scanID, filepath.Base(inputPath))
	fmt.Printf("\n")
}

func inspectPyramid(pyramidPath string) {
	pyramid, err := imagepyramid.GetPyramidInfo(pyramidPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot read pyramid info: %v\n", err)
		return
	}

	fmt.Printf("Pyramid Info:\n")
	fmt.Printf("  Total levels: %d\n", len(pyramid.Pyramid))
	fmt.Printf("  Overall bounds: (%.0f, %.0f) to (%.0f, %.0f)\n",
		pyramid.Bounds.Min.X, pyramid.Bounds.Min.Y,
		pyramid.Bounds.Max.X, pyramid.Bounds.Max.Y)
	fmt.Printf("\n")

	for i, layer := range pyramid.Pyramid {
		width := layer.Bounds.Max.X - layer.Bounds.Min.X
		height := layer.Bounds.Max.Y - layer.Bounds.Min.Y
		fmt.Printf("  Level %d: %.0fx%.0f pixels, %d tiles\n",
			i, width, height, len(layer.Tiles))
	}
}
