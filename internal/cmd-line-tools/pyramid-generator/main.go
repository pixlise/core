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
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/imagepyramid"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var inputPath string
	var outputPath string
	var scanID string
	var imageName string
	var configPath string
	var tileSize int
	var quality int
	var compression string
	var inspect bool

	flag.StringVar(&inputPath, "input", "", "Path to input TIFF file (required)")
	flag.StringVar(&outputPath, "output", "", "Path to output pyramid TIFF (for standalone use)")
	flag.StringVar(&configPath, "config", "", "API config JSON path (for MongoDB/FileAccess integration)")
	flag.StringVar(&scanID, "scanId", "", "Scan ID (required with -config)")
	flag.StringVar(&imageName, "imageName", "", "Image name (e.g., 'z-stack.tif') (required with -config)")
	flag.IntVar(&tileSize, "tile-size", 256, "Tile size in pixels (default: 256)")
	flag.IntVar(&quality, "quality", 85, "JPEG quality 1-100 (default: 85)")
	flag.StringVar(&compression, "compression", "jpeg", "Compression: jpeg or deflate (default: jpeg)")
	flag.BoolVar(&inspect, "inspect", false, "Inspect existing pyramid and show metadata")
	flag.Parse()

	if inputPath == "" {
		printUsage()
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

	// Validate flags for integration mode
	if configPath != "" && (scanID == "" || imageName == "") {
		fmt.Println("ERROR: When using -config, both -scanId and -imageName are required")
		printUsage()
		os.Exit(1)
	}

	// Generate pyramid to temp location
	tempPyramid := filepath.Join(os.TempDir(), fmt.Sprintf("pyramid_%d.tiff", time.Now().UnixNano()))
	defer os.Remove(tempPyramid)

	fmt.Printf("========================================\n")
	fmt.Printf("Pyramidal TIFF Generator\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Input:       %s\n", inputPath)
	if configPath != "" {
		fmt.Printf("Config:      %s\n", configPath)
		fmt.Printf("Scan ID:     %s\n", scanID)
		fmt.Printf("Image name:  %s\n", imageName)
	} else if outputPath != "" {
		fmt.Printf("Output:      %s\n", outputPath)
	}
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

	// Inspect input
	fmt.Printf("Inspecting input TIFF...\n")
	info, err := imagepyramid.InspectTIFF(inputPath)
	if err != nil {
		fmt.Printf("WARNING: Could not inspect TIFF: %v\n", err)
	} else {
		fmt.Printf("  %s\n", info.Print())
		if info.HasPyramid {
			fmt.Printf("\nℹ️  Input already has pyramids! You can use it directly.\n")
		}
		fmt.Printf("========================================\n")
	}

	// Generate pyramid
	fmt.Printf("Generating pyramid... (this may take a while)\n")
	startTime := time.Now()

	genConfig := imagepyramid.GeneratorConfig{
		TileSize:    tileSize,
		Compression: compression,
		Quality:     quality,
	}

	input := imagepyramid.ImageInput{
		Path:    inputPath,
		Channel: 0,
	}

	err = imagepyramid.GeneratePyramidalTIFF(input, tempPyramid, genConfig)
	if err != nil {
		fmt.Printf("ERROR: Pyramid generation failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	tempInfo, err := os.Stat(tempPyramid)
	if err != nil {
		fmt.Printf("ERROR: Cannot stat temp pyramid: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("========================================\n")
	fmt.Printf("✅ Pyramid generated successfully!\n")
	fmt.Printf("========================================\n")
	fmt.Printf("Generation time: %v\n", duration)
	fmt.Printf("Pyramid size:    %.2f MB\n", float64(tempInfo.Size())/(1024*1024))
	fmt.Printf("Compression:     %.1f%% of original\n", float64(tempInfo.Size())/float64(inputInfo.Size())*100)
	fmt.Printf("========================================\n")

	// If config provided, integrate with MongoDB and FileAccess
	if configPath != "" {
		fmt.Printf("\n📦 Integrating with PIXLISE storage...\n")
		err = integrateWithPIXLISE(configPath, scanID, imageName, inputPath, tempPyramid)
		if err != nil {
			fmt.Printf("ERROR: Integration failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ Integration complete!\n")
	}

	// If output path specified, copy there (for backward compatibility)
	if outputPath != "" {
		fmt.Printf("\n📋 Copying to output path...\n")

		// Create output directory
		outputDir := filepath.Dir(outputPath)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("ERROR: Cannot create output directory: %v\n", err)
			os.Exit(1)
		}

		// Copy file
		pyramidBytes, err := os.ReadFile(tempPyramid)
		if err != nil {
			fmt.Printf("ERROR: Cannot read temp pyramid: %v\n", err)
			os.Exit(1)
		}

		err = os.WriteFile(outputPath, pyramidBytes, 0644)
		if err != nil {
			fmt.Printf("ERROR: Cannot write output: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Copied to: %s\n", outputPath)
	}

	// Inspect the generated pyramid
	fmt.Printf("\n📊 Pyramid Metadata:\n")
	inspectPyramid(tempPyramid)

	// Print usage instructions
	fmt.Printf("\n✅ Done!\n")
	if configPath != "" {
		fmt.Printf("\nTo test the API endpoints:\n")
		fmt.Printf("  1. Start API:    go run ./internal/api -customConfigPath=%s\n", configPath)
		fmt.Printf("  2. Get metadata: curl http://localhost:8080/pyramid-info/%s/%s?format=json\n", scanID, imageName)
		fmt.Printf("  3. Get a tile:   curl http://localhost:8080/pyramid-tiles/%s/%s/0/0/0/0 > tile.jpg\n", scanID, imageName)
	}
	fmt.Printf("\n")
}

// integrateWithPIXLISE stores the pyramid in FileAccess and updates MongoDB
func integrateWithPIXLISE(configPath, scanID, imageName, originalImagePath, pyramidTempPath string) error {
	// Load config
	cfg, err := config.NewConfigFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create logger
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	// Connect to MongoDB
	fmt.Printf("  Connecting to MongoDB...\n")
	mongoClient, _, err := mongoDBConnection.Connect(nil, cfg.MongoSecret, iLog)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer mongoClient.Disconnect(context.TODO())

	dbName := mongoDBConnection.GetDatabaseName("pixlise", cfg.EnvironmentName)
	db := mongoClient.Database(dbName)

	// Create FileAccess (use local filesystem for now)
	fmt.Printf("  Using local filesystem storage\n")
	fs := &fileaccess.FSAccess{}

	// Read pyramid bytes
	pyramidBytes, err := os.ReadFile(pyramidTempPath)
	if err != nil {
		return fmt.Errorf("failed to read pyramid: %w", err)
	}

	// Construct storage path
	imagePath := path.Join(scanID, imageName)
	pyramidStoragePath := filepaths.GetPyramidFilePath(imagePath)

	fmt.Printf("  Storing pyramid at: %s\n", pyramidStoragePath)

	// Store pyramid via FileAccess
	err = fs.WriteObject(cfg.DatasetsBucket, pyramidStoragePath, pyramidBytes)
	if err != nil {
		return fmt.Errorf("failed to write pyramid to storage: %w", err)
	}

	fmt.Printf("  ✅ Pyramid stored successfully\n")

	// Get pyramid metadata
	fmt.Printf("  Extracting pyramid metadata...\n")
	pyramidInfo, err := imagepyramid.GetPyramidInfo(pyramidTempPath)
	if err != nil {
		return fmt.Errorf("failed to get pyramid info: %w", err)
	}

	fmt.Printf("  ✅ Metadata extracted (%d levels)\n", len(pyramidInfo.Pyramid))

	// Get image dimensions from pyramid metadata (more reliable than trying to decode TIFF)
	imgWidth := uint32(pyramidInfo.Bounds.Max.X - pyramidInfo.Bounds.Min.X)
	imgHeight := uint32(pyramidInfo.Bounds.Max.Y - pyramidInfo.Bounds.Min.Y)

	stats, err := os.Stat(originalImagePath)
	if err != nil {
		return fmt.Errorf("failed to stat original image: %w", err)
	}

	fmt.Printf("  Image dimensions: %dx%d pixels\n", imgWidth, imgHeight)

	// Create or update ScanImage in MongoDB
	fmt.Printf("  Creating/updating ScanImage in MongoDB...\n")
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ImagesName)

	scanImage := &protos.ScanImage{
		ImagePath:          imagePath,
		Source:             protos.ScanImageSource_SI_UPLOAD,
		Width:              uint32(imgWidth),
		Height:             uint32(imgHeight),
		FileSize:           uint32(stats.Size()),
		Purpose:            protos.ScanImagePurpose_SIP_MULTICHANNEL,
		AssociatedScanIds:  []string{scanID},
		OriginScanId:       scanID,
		OriginImageURL:     "",
		MatchInfo:          nil,
		PyramidDescription: pyramidInfo,
	}

	filter := bson.M{"_id": imagePath}
	update := bson.M{"$set": scanImage}
	opts := options.Update().SetUpsert(true)

	result, err := coll.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("MongoDB update failed: %w", err)
	}

	if result.UpsertedCount > 0 {
		fmt.Printf("  ✅ ScanImage created in MongoDB\n")
	} else if result.ModifiedCount > 0 {
		fmt.Printf("  ✅ ScanImage updated in MongoDB\n")
	} else {
		fmt.Printf("  ℹ️  ScanImage already exists (no changes)\n")
	}

	iLog.Infof("Successfully integrated pyramid for %s", imagePath)
	return nil
}

func inspectPyramid(pyramidPath string) {
	pyramid, err := imagepyramid.GetPyramidInfo(pyramidPath)
	if err != nil {
		fmt.Printf("ERROR: Cannot read pyramid info: %v\n", err)
		return
	}

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

func printUsage() {
	fmt.Println("Error: -input flag is required")
	fmt.Println("")
	fmt.Println("Usage: pyramid-generator -input /path/to/image.tiff [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -input string       Path to input TIFF file (required)")
	fmt.Println("  -config string      API config JSON path (enables MongoDB/FileAccess integration)")
	fmt.Println("  -scanId string      Scan ID (required with -config)")
	fmt.Println("  -imageName string   Image name e.g. 'z-stack.tif' (required with -config)")
	fmt.Println("  -output string      Output path (for standalone use)")
	fmt.Println("  -tile-size int      Tile size in pixels (default: 256)")
	fmt.Println("  -quality int        JPEG quality 1-100 (default: 85)")
	fmt.Println("  -compression string Compression: jpeg or deflate (default: jpeg)")
	fmt.Println("  -inspect            Inspect existing pyramid and show metadata")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Standalone generation (local file only)")
	fmt.Println("  pyramid-generator -input image.tif -output pyramid.tif")
	fmt.Println("")
	fmt.Println("  # Full integration with PIXLISE (FileAccess + MongoDB)")
	fmt.Println("  pyramid-generator -input image.tif -config ./local-api-config.json \\")
	fmt.Println("    -scanId biggerpagesscan -imageName z-stack.tif")
	fmt.Println("")
	fmt.Println("  # Inspect existing pyramid")
	fmt.Println("  pyramid-generator -input pyramid.tiff -inspect")
}
