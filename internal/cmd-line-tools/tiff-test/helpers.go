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

import "github.com/cshum/vipsgen/vips"

func interpretationName(i vips.Interpretation) string {
	switch i {
	case vips.InterpretationMultiband:
		return "multiband"
	case vips.InterpretationBW:
		return "b-w"
	case vips.InterpretationHistogram:
		return "histogram"
	case vips.InterpretationXyz:
		return "xyz"
	case vips.InterpretationLab:
		return "lab"
	case vips.InterpretationCmyk:
		return "cmyk"
	case vips.InterpretationLabq:
		return "labq"
	case vips.InterpretationRgb:
		return "rgb"
	case vips.InterpretationCmc:
		return "cmc"
	case vips.InterpretationLch:
		return "lch"
	case vips.InterpretationLabs:
		return "labs"
	case vips.InterpretationSrgb:
		return "srgb"
	case vips.InterpretationYxy:
		return "yxy"
	case vips.InterpretationFourier:
		return "fourier"
	case vips.InterpretationRgb16:
		return "rgb16"
	case vips.InterpretationGrey16:
		return "grey16"
	case vips.InterpretationMatrix:
		return "matrix"
	case vips.InterpretationScrgb:
		return "scrgb"
	case vips.InterpretationHsv:
		return "hsv"
	default:
		return "unknown"
	}
}
