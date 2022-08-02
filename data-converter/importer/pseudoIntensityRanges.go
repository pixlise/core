// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package importer

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/pixlise/core/core/logger"
	"github.com/pixlise/core/core/utils"
	"github.com/pixlise/core/data-converter/converterModels"
)

func ReadPseudoIntensityRangesFile(path string, jobLog logger.ILogger) ([]converterModels.PseudoIntensityRange, error) {
	data, err := ReadCSV(path, 0, ',', jobLog)
	if err != nil {
		return nil, err
	}

	return parseRanges(data)
}

func parseRanges(data [][]string) ([]converterModels.PseudoIntensityRange, error) {
	expHeaders := []string{"Name", "StartChannel", "EndChannel"}
	if !utils.StringSlicesEqual(expHeaders, data[0]) {
		return nil, errors.New("Pseudo-intensity ranges has unexpected headers")
	}

	result := []converterModels.PseudoIntensityRange{}

	for idx, row := range data[1:] {
		s, err := strconv.Atoi(row[1])
		if err != nil {
			return nil, fmt.Errorf("Failed to read start value from row %v of pseudointensity range file. Got: \"%v\"", idx, row)
		}
		t, err := strconv.Atoi(row[2])
		if err != nil {
			return nil, fmt.Errorf("Failed to read end value from row %v of pseudointensity range file. Got: \"%v\"", idx, row)
		}

		r := converterModels.PseudoIntensityRange{
			Name:  row[0],
			Start: s,
			End:   t,
		}
		result = append(result, r)
	}

	return result, nil
}
