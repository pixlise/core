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
