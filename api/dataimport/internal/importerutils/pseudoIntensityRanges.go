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

package importerutils

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pixlise/core/v3/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/utils"
)

func ReadPseudoIntensityRangesFile(path string, jobLog logger.ILogger) ([]dataConvertModels.PseudoIntensityRange, error) {
	data, err := ReadCSV(path, 0, ',', jobLog)
	if err != nil {
		return nil, err
	}

	return parseRanges(data)
}

func parseRanges(data [][]string) ([]dataConvertModels.PseudoIntensityRange, error) {
	expHeaders := []string{"Name", "StartChannel", "EndChannel"}
	if !utils.SlicesEqual(expHeaders, data[0]) {
		return nil, errors.New("Pseudo-intensity ranges has unexpected headers")
	}

	result := []dataConvertModels.PseudoIntensityRange{}

	for idx, row := range data[1:] {
		s, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, fmt.Errorf("Failed to read start value from row %v of pseudointensity range file. Got: \"%v\"", idx, row)
		}
		t, err := strconv.Atoi(row[2])
		if err != nil {
			return nil, fmt.Errorf("Failed to read end value from row %v of pseudointensity range file. Got: \"%v\"", idx, row)
		}

		r := dataConvertModels.PseudoIntensityRange{
			Name:  row[0],
			Start: s,
			End:   t,
		}
		result = append(result, r)
	}

	return result, nil
}
