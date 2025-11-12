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
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/chai2010/tiff"
	_ "golang.org/x/image/tiff"
)

// TIFF IFD tag constants
const (
	tagSubIFD         = 330
	tagImageWidth     = 256
	tagImageHeight    = 257
	tagBitsPerSample  = 258
	tagCompression    = 259
	tagPhotometric    = 262
	tagStripOffsets   = 273
	tagSamplesPerPixel = 277
	tagRowsPerStrip   = 278
	tagStripByteCounts = 279
	tagTileWidth      = 322
	tagTileLength     = 323
	tagTileOffsets    = 324
	tagTileByteCounts = 325
)

// TIFF data type constants
const (
	typeByte      = 1
	typeASCII     = 2
	typeShort     = 3
	typeLong      = 4
	typeRational  = 5
	typeSByte     = 6
	typeUndefined = 7
	typeSShort    = 8
	typeSLong     = 9
	typeSRational = 10
	typeFloat     = 11
	typeDouble    = 12
	typeIFD       = 13
	typeLong8     = 16
	typeSLong8    = 17
	typeIFD8      = 18
)

// IFDInfo contains information about a TIFF IFD
type IFDInfo struct {
	Offset      int64
	Width       uint32
	Height      uint32
	IsTiled     bool
	TileWidth   uint32
	TileHeight  uint32
	Compression uint32
	SubIFDs     []int64
}

// TIFFInfo contains basic TIFF file information
type TIFFInfo struct {
	ByteOrder   binary.ByteOrder
	IsBigTIFF   bool
	FirstIFD    int64
}

// getByteOrder reads and returns the byte order of a TIFF file
func getByteOrder(file *os.File) (binary.ByteOrder, error) {
	info, err := getTIFFInfo(file)
	if err != nil {
		return nil, err
	}
	return info.ByteOrder, nil
}

// getTIFFInfo reads TIFF header and returns file information
func getTIFFInfo(file *os.File) (*TIFFInfo, error) {
	file.Seek(0, 0)
	b := make([]byte, 16)
	if _, err := io.ReadFull(file, b); err != nil {
		return nil, err
	}

	var byteOrder binary.ByteOrder
	if b[0] == 0x4D && b[1] == 0x4D {
		byteOrder = binary.BigEndian
	} else if b[0] == 0x49 && b[1] == 0x49 {
		byteOrder = binary.LittleEndian
	} else {
		return nil, fmt.Errorf("invalid TIFF byte order marker")
	}

	// Check magic number
	magic := byteOrder.Uint16(b[2:4])

	info := &TIFFInfo{
		ByteOrder: byteOrder,
	}

	if magic == 42 {
		// Standard TIFF
		info.IsBigTIFF = false
		info.FirstIFD = int64(byteOrder.Uint32(b[4:8]))
	} else if magic == 43 {
		// BigTIFF
		info.IsBigTIFF = true
		// BigTIFF has: offsetsize (2 bytes), always 0 (2 bytes), then 64-bit IFD offset
		offsetSize := byteOrder.Uint16(b[4:6])
		if offsetSize != 8 {
			return nil, fmt.Errorf("BigTIFF with unexpected offset size: %d", offsetSize)
		}
		info.FirstIFD = int64(byteOrder.Uint64(b[8:16]))
	} else {
		return nil, fmt.Errorf("invalid TIFF magic number: %d", magic)
	}

	return info, nil
}

// readIFDInfo reads detailed information from an IFD at the given offset
func readIFDInfo(file *os.File, offset int64, byteOrder binary.ByteOrder, isBigTIFF bool, verbose bool) (*IFDInfo, error) {
	info := &IFDInfo{
		Offset:  offset,
		SubIFDs: make([]int64, 0),
	}

	// Seek to IFD
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, err
	}

	// Read number of directory entries
	var numEntries uint64
	if isBigTIFF {
		if err := binary.Read(file, byteOrder, &numEntries); err != nil {
			return nil, err
		}
	} else {
		var numEntries16 uint16
		if err := binary.Read(file, byteOrder, &numEntries16); err != nil {
			return nil, err
		}
		numEntries = uint64(numEntries16)
	}

	if verbose {
		fmt.Printf("  Reading IFD at offset %d with %d entries\n", offset, numEntries)
	}

	// Sanity check on number of entries
	if numEntries > 1000 {
		if verbose {
			fmt.Printf("  WARNING: Unusually large number of entries (%d), skipping detailed parsing\n", numEntries)
		}
		// Skip this IFD's detailed parsing
		return info, nil
	}

	// Read each entry
	for i := uint64(0); i < numEntries; i++ {
		var tag, fieldType uint16
		var count uint64
		var valueOffset uint64

		if err := binary.Read(file, byteOrder, &tag); err != nil {
			return nil, err
		}
		if err := binary.Read(file, byteOrder, &fieldType); err != nil {
			return nil, err
		}

		if isBigTIFF {
			// BigTIFF: 8-byte count and 8-byte value/offset
			if err := binary.Read(file, byteOrder, &count); err != nil {
				return nil, err
			}
			if err := binary.Read(file, byteOrder, &valueOffset); err != nil {
				return nil, err
			}
		} else {
			// Standard TIFF: 4-byte count and 4-byte value/offset
			var count32 uint32
			var valueOffset32 uint32
			if err := binary.Read(file, byteOrder, &count32); err != nil {
				return nil, err
			}
			if err := binary.Read(file, byteOrder, &valueOffset32); err != nil {
				return nil, err
			}
			count = uint64(count32)
			valueOffset = uint64(valueOffset32)
		}

		// Extract relevant tag values
		// For simple numeric tags where value fits in the offset field
		switch tag {
		case tagImageWidth:
			if count == 1 {
				info.Width = uint32(valueOffset)
			}
		case tagImageHeight:
			if count == 1 {
				info.Height = uint32(valueOffset)
			}
		case tagTileWidth:
			info.IsTiled = true
			if count == 1 {
				info.TileWidth = uint32(valueOffset)
			}
		case tagTileLength:
			if count == 1 {
				info.TileHeight = uint32(valueOffset)
			}
		case tagCompression:
			if count == 1 {
				info.Compression = uint32(valueOffset)
			}
		case tagSubIFD:
			// SubIFD tag contains offsets to sub-IFDs (pyramid levels)
			// Limit count to prevent hanging on corrupted files
			if count > 0 && count <= 100 {
				// Save current position
				currentPos, _ := file.Seek(0, 1)

				// SubIFD offsets are stored at the valueOffset location
				file.Seek(int64(valueOffset), 0)

				if isBigTIFF {
					for j := uint64(0); j < count; j++ {
						var subOffset uint64
						if err := binary.Read(file, byteOrder, &subOffset); err == nil {
							info.SubIFDs = append(info.SubIFDs, int64(subOffset))
						}
					}
				} else {
					for j := uint64(0); j < count; j++ {
						var subOffset uint32
						if err := binary.Read(file, byteOrder, &subOffset); err == nil {
							info.SubIFDs = append(info.SubIFDs, int64(subOffset))
						}
					}
				}

				// Restore position
				file.Seek(currentPos, 0)
			}
		}
	}

	return info, nil
}

// analyzeIFDStructure analyzes all IFDs in the TIFF file, including SubIFDs
func analyzeIFDStructure(file *os.File, verbose bool) ([]IFDInfo, error) {
	// Get file size for sanity checks
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	if verbose {
		fmt.Printf("File size: %d bytes\n", fileSize)
	}

	// Get TIFF info (byte order, BigTIFF flag, first IFD offset)
	tiffInfo, err := getTIFFInfo(file)
	if err != nil {
		return nil, err
	}

	if verbose {
		if tiffInfo.ByteOrder == binary.LittleEndian {
			fmt.Printf("Byte order: Little Endian\n")
		} else {
			fmt.Printf("Byte order: Big Endian\n")
		}
		if tiffInfo.IsBigTIFF {
			fmt.Printf("Format: BigTIFF\n")
		} else {
			fmt.Printf("Format: Standard TIFF\n")
		}
		fmt.Printf("First IFD offset: %d (0x%x)\n", tiffInfo.FirstIFD, tiffInfo.FirstIFD)
	}

	var ifdInfos []IFDInfo
	visitedOffsets := make(map[int64]bool)
	maxIFDs := 1000 // Safety limit
	ifdOffset := tiffInfo.FirstIFD

	// Walk through main IFD chain
	for ifdOffset != 0 && ifdOffset < fileSize && len(ifdInfos) < maxIFDs {
		// Check for circular references
		if visitedOffsets[ifdOffset] {
			if verbose {
				fmt.Printf("WARNING: Circular IFD reference detected at offset %d\n", ifdOffset)
			}
			break
		}
		visitedOffsets[ifdOffset] = true

		if verbose {
			fmt.Printf("Processing IFD #%d at offset %d (0x%x)\n", len(ifdInfos), ifdOffset, ifdOffset)
		}

		// Read this IFD's info
		info, err := readIFDInfo(file, ifdOffset, tiffInfo.ByteOrder, tiffInfo.IsBigTIFF, verbose)
		if err != nil {
			if verbose {
				fmt.Printf("ERROR reading IFD: %v\n", err)
			}
			break
		}
		ifdInfos = append(ifdInfos, *info)

		// Seek to end of this IFD to get next offset
		file.Seek(ifdOffset, 0)

		// Read number of entries (to skip them)
		var numEntries uint64
		if tiffInfo.IsBigTIFF {
			if err := binary.Read(file, tiffInfo.ByteOrder, &numEntries); err != nil {
				if verbose {
					fmt.Printf("ERROR reading numEntries: %v\n", err)
				}
				break
			}
			// Skip entries (20 bytes each in BigTIFF)
			file.Seek(int64(numEntries)*20, 1)

			// Read next IFD offset (8 bytes in BigTIFF)
			var nextOffset uint64
			if err := binary.Read(file, tiffInfo.ByteOrder, &nextOffset); err != nil {
				if verbose {
					fmt.Printf("ERROR reading next offset: %v\n", err)
				}
				break
			}
			ifdOffset = int64(nextOffset)
		} else {
			var numEntries16 uint16
			if err := binary.Read(file, tiffInfo.ByteOrder, &numEntries16); err != nil {
				if verbose {
					fmt.Printf("ERROR reading numEntries: %v\n", err)
				}
				break
			}
			numEntries = uint64(numEntries16)

			// Skip entries (12 bytes each in standard TIFF)
			file.Seek(int64(numEntries)*12, 1)

			// Read next IFD offset (4 bytes in standard TIFF)
			var nextOffset uint32
			if err := binary.Read(file, tiffInfo.ByteOrder, &nextOffset); err != nil {
				if verbose {
					fmt.Printf("ERROR reading next offset: %v\n", err)
				}
				break
			}
			ifdOffset = int64(nextOffset)
		}

		if verbose && ifdOffset != 0 {
			fmt.Printf("Next IFD offset: %d (0x%x)\n", ifdOffset, ifdOffset)
		}
	}

	if verbose {
		fmt.Printf("Total IFDs found: %d\n", len(ifdInfos))
	}

	return ifdInfos, nil
}

// countMainIFDs counts the number of main IFD directories (excluding SubIFDs)
func countMainIFDs(file *os.File) (int, error) {
	ifdInfos, err := analyzeIFDStructure(file, false)
	if err != nil {
		return 0, err
	}
	return len(ifdInfos), nil
}

// decodeIFDAtOffset decodes a single TIFF IFD at a specific offset
func decodeIFDAtOffset(filePath string, offset int64) (image.Image, error) {
	// For now, we'll use the chai2010 library but only decode single pages
	// This is a simplified version - a full implementation would seek to the offset
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Use tiff.Decode which only decodes the first IFD
	// To decode specific IFDs, we'd need to modify the library or use a different approach
	img, err := tiff.Decode(file)
	return img, err
}

func main() {
	var filePath string
	var verbose bool

	flag.StringVar(&filePath, "file", "", "Path to TIFF file to test")
	flag.BoolVar(&verbose, "v", false, "Verbose output (show detailed IFD parsing)")
	flag.Parse()

	if filePath == "" {
		fmt.Println("Error: -file flag is required")
		fmt.Println("Usage: tiff-test -file /path/to/image.tiff [-v]")
		os.Exit(1)
	}

	fmt.Printf("========================================\n")
	fmt.Printf("TIFF Image Loading Test\n")
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

	// Open the file
	fmt.Printf("Opening file...\n")
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("ERROR: Cannot open file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Analyze TIFF structure including SubIFDs
	fmt.Printf("Analyzing TIFF structure...\n")
	if verbose {
		fmt.Printf("(Verbose mode enabled)\n")
	}
	ifdInfos, err := analyzeIFDStructure(file, verbose)
	if err != nil {
		fmt.Printf("ERROR: Cannot parse TIFF structure: %v\n", err)
		os.Exit(1)
	}

	numIFDs := len(ifdInfos)
	totalSubIFDs := 0
	for _, info := range ifdInfos {
		totalSubIFDs += len(info.SubIFDs)
	}

	fmt.Printf("Found %d main IFD(s) in file\n", numIFDs)
	if totalSubIFDs > 0 {
		fmt.Printf("Found %d SubIFD(s) total (pyramid levels)\n", totalSubIFDs)
	}

	// Display structure details
	for i, info := range ifdInfos {
		compressionName := "Unknown"
		switch info.Compression {
		case 1:
			compressionName = "None"
		case 5:
			compressionName = "LZW"
		case 7:
			compressionName = "JPEG"
		case 8:
			compressionName = "Deflate"
		case 32773:
			compressionName = "PackBits"
		}

		fmt.Printf("  IFD %d: %dx%d", i, info.Width, info.Height)
		if info.IsTiled {
			fmt.Printf(" (tiled %dx%d)", info.TileWidth, info.TileHeight)
		}
		fmt.Printf(" Compression: %s", compressionName)
		if len(info.SubIFDs) > 0 {
			fmt.Printf(" [%d pyramid levels]", len(info.SubIFDs))
		}
		fmt.Printf("\n")
	}

	if numIFDs > 1 || totalSubIFDs > 0 {
		fmt.Printf("(This is a multi-page/multi-layer TIFF with z-stack or pyramid structure)\n")
	}
	fmt.Printf("----------------------------------------\n")

	// Get memory stats before loading
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	fmt.Printf("Memory before load: %.2f MB\n", float64(memBefore.Alloc)/(1024*1024))

	// Decode ONLY the first page to avoid hanging on massive files
	fmt.Printf("Decoding first page only (to avoid loading all pyramid levels)...\n")
	startTime := time.Now()

	var img image.Image
	var format string
	var loadDuration time.Duration

	// Reset file position
	file.Seek(0, 0)

	// Try chai2010/tiff library (supports JPEG compression)
	// Use Decode instead of DecodeAll to only load the first IFD
	img, err = tiff.Decode(file)

	if err != nil {
		// Fall back to standard image.Decode
		file.Seek(0, 0)
		img, format, err = image.Decode(file)
		if err != nil {
			fmt.Printf("ERROR: Cannot decode image: %v\n", err)
			fmt.Printf("\nTroubleshooting:\n")
			fmt.Printf("- This TIFF may use unsupported compression (e.g., JPEG compression)\n")
			fmt.Printf("- For JPEG-compressed TIFFs, ensure github.com/chai2010/tiff is properly installed\n")
			fmt.Printf("- Run: go get github.com/chai2010/tiff\n")
			os.Exit(1)
		}
	} else {
		format = "tiff"
	}

	loadDuration = time.Since(startTime)

	// Get memory stats after loading
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Get image info
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	colorModel := img.ColorModel()

	// Print results
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("SUCCESS!\n")
	fmt.Printf("----------------------------------------\n")
	fmt.Printf("Format: %s\n", format)
	if numIFDs > 1 {
		fmt.Printf("NOTE: Only decoded page 1 of %d\n", numIFDs)
		fmt.Printf("(Other pages may contain z-slices, channels, or pyramid levels)\n")
	}
	fmt.Printf("Dimensions: %d x %d pixels\n", width, height)
	fmt.Printf("Total pixels: %d (%.2f megapixels)\n", width*height, float64(width*height)/1000000)
	fmt.Printf("Color model: %T\n", colorModel)
	fmt.Printf("Load time: %v\n", loadDuration)
	fmt.Printf("Memory after load: %.2f MB\n", float64(memAfter.Alloc)/(1024*1024))
	fmt.Printf("Memory increase: %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/(1024*1024))
	fmt.Printf("========================================\n")

	// Sample a pixel from the center to verify we can access the data
	centerX := width / 2
	centerY := height / 2
	centerPixel := img.At(centerX, centerY)
	r, g, b, a := centerPixel.RGBA()
	fmt.Printf("Center pixel (%d, %d): R=%d G=%d B=%d A=%d\n", centerX, centerY, r>>8, g>>8, b>>8, a>>8)
	fmt.Printf("========================================\n")
}
