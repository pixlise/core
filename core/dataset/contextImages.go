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

package dataset

// MatchedAlignedImageMeta - metadata for an image that's transformed to match an AlignedImage (eg MCC)
type MatchedAlignedImageMeta struct {
	// PMC of the MCC image whose beam locations this image is matched with
	AlignedBeamPMC int32 `json:"aligned-beam-pmc"`

	// File name of the matched image - the one that was imported with an area matching the Aligned image
	MatchedImageName string `json:"matched-image"`

	// This is the x/y offset of the sub-image area where the Matched image matches the Aligned image
	// In other words, the top-left Aligned image pixel is at (XOffset, YOffset) in the matched image
	XOffset float32 `json:"x-offset"`
	YOffset float32 `json:"y-offset"`

	// The relative sizing of the sub-image area where the Matched image matches the Aligned image
	// In other words, if the Aligned image is 752x580 pixels, and the Matched image is much higher res
	// at 2000x3000, and within that a central area of 1600x1300, scale is (1600/752, 1300/580) = (2.13, 2.24)
	XScale float32 `json:"x-scale"`
	YScale float32 `json:"y-scale"`

	// Full path, no JSON field because this is only used internally during dataset conversion
	MatchedImageFullPath string `json:"-"`
}
