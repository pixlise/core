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

package pixlem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v4/api/dataimport/internal/converters/pixlfm"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// These are EM files, which we expect to be in the same format as FM but because they come from
// manual uploads, we expect the actual files to be in a sub dir. We also override the group/detector
// when importing these

type PIXLEM struct {
}

func (p PIXLEM) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	// Find the subdir
	subdir := ""

	c, _ := os.ReadDir(importPath)
	for _, entry := range c {
		if entry.IsDir() {
			// If it's not the first one, we can't do this
			if len(subdir) > 0 {
				return nil, "", fmt.Errorf("Found multiple subdirs (\"%v\", \"%v\"), expected one in: \"%v\"", subdir, entry.Name(), importPath)
			}
			subdir = entry.Name()
		}
	}

	if len(subdir) <= 0 {
		return nil, "", errors.New("Failed to find PIXL data subdir in: " + importPath)
	}

	// Form the actual path to the files
	subImportPath := filepath.Join(importPath, subdir)
	fmImporter := pixlfm.PIXLFM{}

	// Override importers group and detector
	fmImporter.SetOverrides(protos.ScanInstrument_PIXL_EM, "PIXL-EM-E2E")

	// Now we can import it like normal
	return fmImporter.Import(subImportPath, pseudoIntensityRangesPath, datasetIDExpected, log)
}
