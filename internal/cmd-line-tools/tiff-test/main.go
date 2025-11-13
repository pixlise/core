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
	"runtime"
	"time"

	"github.com/cshum/vipsgen/vips"
)

func main() {
	var filePath string
	var verbose bool
	var pageNum int
	var outputPath string

	flag.StringVar(&filePath, "file", "", "Path to TIFF file to test")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.IntVar(&pageNum, "page", 0, "Page number to decode (0-based, default: 0)")
	flag.StringVar(&outputPath, "output", "", "Optional: save extracted page to this path")
	flag.Parse()

	if filePath == "" {
		fmt.Println("Error: -file flag is required")
		fmt.Println("Usage: tiff-test -file /path/to/image.tiff [-v] [-page N] [-output path.png]")
		fmt.Println("")
		fmt.Println("Options:")
		fmt.Println("  -file string      Path to TIFF file")
		fmt.Println("  -v                Verbose output")
		fmt.Println("  -page int         Page/IFD number to decode (0-based, default: 0)")
		fmt.Println("  -output string    Save extracted page to output file")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  tiff-test -file image.tif")
		fmt.Println("  tiff-test -file image.tif -page 3")
		fmt.Println("  tiff-test -file image.tif -page 2 -output page2.png")
		os.Exit(1)
	}

	fmt.Printf("========================================\n")
	fmt.Printf("TIFF Image Loading Test (using vipsgen)\n")
	fmt.Printf("========================================\n")
	fmt.Printf("File: %s\n", filePath)
	fmt.Printf("----------------------------------------\n")

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("ERROR: Cannot access file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File size: %.2f MB (%d bytes)\n", float64(fileInfo.Size())/(1024*1024), fileInfo.Size())
	fmt.Printf("----------------------------------------\n")

	// Get memory stats before loading
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	fmt.Printf("Memory before load: %.2f MB\n", float64(memBefore.Alloc)/(1024*1024))

	// Load the TIFF page
	fmt.Printf("Loading TIFF (page %d)...\n", pageNum)
	startTime := time.Now()

	img, err := vips.NewTiffload(filePath, &vips.TiffloadOptions{
		Page: pageNum,
		N:    1, // Load only 1 page
	})
	if err != nil {
		fmt.Printf("ERROR: Cannot load TIFF: %v\n", err)
		fmt.Printf("\nTips:\n")
		fmt.Printf("- Ensure libvips is installed (brew install vips or apt install libvips-dev)\n")
		fmt.Printf("- Check that the page number is valid for this TIFF\n")
		os.Exit(1)
	}
	defer img.Close()

	loadDuration := time.Since(startTime)

	// Get memory stats after loading
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Get image info
	width := img.Width()
	height := img.Height()
	bands := img.Bands()
	pages := img.Pages()

	// Print results
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("SUCCESS!\n")
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("Dimensions: %d x %d pixels\n", width, height)
	fmt.Printf("Total pixels: %d (%.2f megapixels)\n", width*height, float64(width*height)/1000000)
	fmt.Printf("Bands: %d\n", bands)
	fmt.Printf("Format: %v\n", img.Format())
	fmt.Printf("Interpretation: %v\n", img.Interpretation())
	if pages > 1 {
		fmt.Printf("Total pages in file: %d (loaded page %d)\n", pages, pageNum)
	}
	fmt.Printf("Load time: %v\n", loadDuration)
	fmt.Printf("Memory after load: %.2f MB\n", float64(memAfter.Alloc)/(1024*1024))
	fmt.Printf("Memory increase: %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))

	if verbose {
		// Resolution info not directly available in vipsgen API
		fmt.Printf("Coding: %v\n", img.Coding())
	}

	fmt.Printf("========================================\n")

	// Sample pixel from center to verify we can access data
	if width > 0 && height > 0 {
		centerX := width / 2
		centerY := height / 2
		pixel, err := img.Getpoint(centerX, centerY, nil)
		if err == nil {
			fmt.Printf("Center pixel (%d, %d): ", centerX, centerY)
			for i, val := range pixel {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("Band%d=%.0f", i, val)
			}
			fmt.Printf("\n")
			fmt.Printf("========================================\n")
		}
	}

	// Save to output file if requested
	if outputPath != "" {
		fmt.Printf("Saving to %s...\n", outputPath)
		startTime := time.Now()

		// Use Pngsave for PNG output, Jpegsave for JPEG, etc.
		// img methods are named after the libvips save operations
		err = img.Pngsave(outputPath, nil)
		if err != nil {
			fmt.Printf("ERROR: Cannot save output: %v\n", err)
			os.Exit(1)
		}

		saveDuration := time.Since(startTime)

		outInfo, err := os.Stat(outputPath)
		if err == nil {
			fmt.Printf("Saved successfully in %v\n", saveDuration)
			fmt.Printf("Output size: %.2f MB\n", float64(outInfo.Size())/(1024*1024))
		}
		fmt.Printf("========================================\n")
	}
}
