// // Licensed to NASA JPL under one or more contributor
// // license agreements. See the NOTICE file distributed with
// // this work for additional information regarding copyright
// // ownership. NASA JPL licenses this file to you under
// // the Apache License, Version 2.0 (the "License"); you may
// // not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //     http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing,
// // software distributed under the License is distributed on an
// // "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// // KIND, either express or implied.  See the License for the
// // specific language governing permissions and limitations
// // under the License.

// package endpoints

// import (
// 	"fmt"
// 	"os"
// 	"path/filepath"
// 	"strconv"

// 	apiRouter "github.com/pixlise/core/v4/api/router"
// 	"github.com/pixlise/core/v4/core/errorwithstatus"
// )

// // GetPyramidTileSimple serves tiles from local DeepZoom files (no S3, no caching, no permissions for now)
// // URL: /pyramid-tiles/{scan}/{filename}/{page}/{level}/{x}/{y}
// // Reads from: ~/PIXLISE/TESTING/{scan}/{filename}/page_{page}_files/{level}/{x}_{y}.jpg
// func GetPyramidTileSimple(params apiRouter.ApiHandlerGenericPublicParams) error {
// 	// Parse path parameters
// 	scanID := params.PathParams[ScanIdentifier]
// 	fileName := params.PathParams[FileNameIdentifier]

// 	page, err := strconv.Atoi(params.PathParams[PageIdentifier])
// 	if err != nil {
// 		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid page: %v", params.PathParams[PageIdentifier]))
// 	}

// 	level, err := strconv.Atoi(params.PathParams[LevelIdentifier])
// 	if err != nil {
// 		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid level: %v", params.PathParams[LevelIdentifier]))
// 	}

// 	x, err := strconv.Atoi(params.PathParams[TileXIdentifier])
// 	if err != nil {
// 		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid x: %v", params.PathParams[TileXIdentifier]))
// 	}

// 	y, err := strconv.Atoi(params.PathParams[TileYIdentifier])
// 	if err != nil {
// 		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid y: %v", params.PathParams[TileYIdentifier]))
// 	}

// 	// Construct path to DeepZoom tile
// 	// Structure: ~/PIXLISE/TESTING/{scanID}/{fileName}/page_{page}_files/{level}/{x}_{y}.jpg
// 	tilePath := getDeepZoomTilePath(scanID, fileName, page, level, x, y)

// 	// Check if tile exists
// 	if _, err := os.Stat(tilePath); os.IsNotExist(err) {
// 		return errorwithstatus.MakeNotFoundError(fmt.Sprintf("tile not found: %s", tilePath))
// 	}

// 	// Read tile file
// 	tileBytes, err := os.ReadFile(tilePath)
// 	if err != nil {
// 		return fmt.Errorf("failed to read tile: %w", err)
// 	}

// 	// Write JPEG bytes to response
// 	params.Writer.Header().Set("Content-Type", "image/jpeg")
// 	params.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(tileBytes)))
// 	_, err = params.Writer.Write(tileBytes)
// 	return err
// }

// // GetPyramidInfoSimple returns ImagePyramid metadata from local DeepZoom .dzi files
// // URL: /pyramid-info/{scan}/{filename}
// func GetPyramidInfoSimple(params apiRouter.ApiHandlerGenericPublicParams) error {
// 	// For now, just return a basic error since we'd need to parse .dzi XML files
// 	// This can be implemented later if needed
// 	return errorwithstatus.MakeNotFoundError("pyramid info endpoint not yet implemented for DeepZoom tiles")
// }

// // getDeepZoomTilePath constructs the local filesystem path for a DeepZoom tile
// // Path: ~/PIXLISE/TESTING/{scanID}/{fileName}/page_{page}_files/{level}/{x}_{y}.jpg
// func getDeepZoomTilePath(scanID, fileName string, page, level, x, y int) string {
// 	homeDir, err := os.UserHomeDir()
// 	if err != nil {
// 		// Fallback to /tmp if home dir unavailable
// 		homeDir = "/tmp"
// 	}

// 	// Structure matches what pyramid-generator creates
// 	tilePath := filepath.Join(
// 		homeDir,
// 		"PIXLISE",
// 		"TESTING",
// 		scanID,
// 		fileName,
// 		fmt.Sprintf("page_%d_files", page),
// 		fmt.Sprintf("%d", level),
// 		fmt.Sprintf("%d_%d.jpg", x, y),
// 	)

// 	return tilePath
// }
