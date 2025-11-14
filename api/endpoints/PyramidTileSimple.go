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

package endpoints

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/imagepyramid"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
)

// GetPyramidTileSimple serves tiles from local pyramid files (no S3, no caching, no permissions for now)
// URL: /pyramid-tiles/{scan}/{filename}/{page}/{level}/{x}/{y}
func GetPyramidTileSimple(params apiRouter.ApiHandlerGenericPublicParams) error {
	// Parse path parameters
	scanID := params.PathParams[ScanIdentifier]
	fileName := params.PathParams[FileNameIdentifier]

	page, err := strconv.Atoi(params.PathParams[PageIdentifier])
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid page: %v", params.PathParams[PageIdentifier]))
	}

	level, err := strconv.Atoi(params.PathParams[LevelIdentifier])
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid level: %v", params.PathParams[LevelIdentifier]))
	}

	x, err := strconv.Atoi(params.PathParams[TileXIdentifier])
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid x: %v", params.PathParams[TileXIdentifier]))
	}

	y, err := strconv.Atoi(params.PathParams[TileYIdentifier])
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("invalid y: %v", params.PathParams[TileYIdentifier]))
	}

	// Construct local pyramid path
	pyramidPath := getPyramidPath(scanID, fileName)

	// Check if pyramid exists
	if _, err := os.Stat(pyramidPath); os.IsNotExist(err) {
		return errorwithstatus.MakeNotFoundError(fmt.Sprintf("pyramid not found: %s", pyramidPath))
	}

	// Extract tile directly from pyramid (handles multi-page)
	tileBytes, err := imagepyramid.ExtractTileFromPage(pyramidPath, page, level, x, y, 256)
	if err != nil {
		// If it's an out-of-bounds error (invalid page/level/tile), return 404
		// Otherwise return 500
		errStr := err.Error()
		if strings.Contains(errStr, "out of bounds") || strings.Contains(errStr, "does not exist") {
			return errorwithstatus.MakeNotFoundError(errStr)
		}
		return fmt.Errorf("failed to extract tile: %w", err)
	}

	// Write JPEG bytes to response
	params.Writer.Header().Set("Content-Type", "image/jpeg")
	params.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(tileBytes)))
	_, err = params.Writer.Write(tileBytes)
	return err
}

// GetPyramidInfoSimple returns ImagePyramid metadata from local pyramid files
// URL: /pyramid-info/{scan}/{filename}
func GetPyramidInfoSimple(params apiRouter.ApiHandlerGenericPublicParams) error {
	scanID := params.PathParams[ScanIdentifier]
	fileName := params.PathParams[FileNameIdentifier]

	// Construct local pyramid path
	pyramidPath := getPyramidPath(scanID, fileName)

	// Check if pyramid exists
	if _, err := os.Stat(pyramidPath); os.IsNotExist(err) {
		return errorwithstatus.MakeNotFoundError(fmt.Sprintf("pyramid not found: %s", pyramidPath))
	}

	// Get pyramid metadata
	pyramidInfo, err := imagepyramid.GetPyramidInfo(pyramidPath)
	if err != nil {
		return fmt.Errorf("failed to read pyramid info: %w", err)
	}

	// Check if user wants JSON (via query param or Accept header)
	wantsJSON := params.Request.URL.Query().Get("format") == "json" ||
		params.Request.Header.Get("Accept") == "application/json"

	if wantsJSON {
		// Return JSON response
		utils.SendProtoJSON(params.Writer, pyramidInfo)
	} else {
		// Return protobuf response
		utils.SendProtoBinary(params.Writer, pyramidInfo)
	}
	return nil
}

// getPyramidPath constructs the local filesystem path for a pyramid TIFF
// Path: ~/PIXLISE/Pyramids/{scanId}/{fileNameWithoutExt}/pyramid.tiff
func getPyramidPath(scanID, fileName string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to /tmp if home dir unavailable
		homeDir = "/tmp"
	}

	// Remove extension from fileName to create subdirectory
	ext := filepath.Ext(fileName)
	nameWithoutExt := fileName[:len(fileName)-len(ext)]

	return filepath.Join(homeDir, "PIXLISE", "Pyramids", scanID, nameWithoutExt, "pyramid.tiff")
}
