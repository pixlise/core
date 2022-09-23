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

package dataConvertModels

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	protos "github.com/pixlise/core/v2/generated-protos"
)

// Importing code needs to store everything in these intermediate models, which are then understood by the output
// code that writes the PIXLISE binary files

// MetaValue - A variant to store an individual metadata value
type MetaValue struct {
	SValue   string
	IValue   int32
	FValue   float32
	DataType protos.Experiment_MetaDataType
}

// StringMetaValue - short-hand for creating a string metadata value variant
func StringMetaValue(s string) MetaValue {
	return MetaValue{SValue: s, DataType: protos.Experiment_MT_STRING}
}

// IntMetaValue - short-hand for creating a int metadata value variant
func IntMetaValue(i int32) MetaValue {
	return MetaValue{IValue: i, DataType: protos.Experiment_MT_INT}
}

// FloatMetaValue - short-hand for creating a float metadata value variant
func FloatMetaValue(f float32) MetaValue {
	return MetaValue{FValue: f, DataType: protos.Experiment_MT_FLOAT}
}

// MetaData - Map of label->meta value variant
type MetaData map[string]MetaValue

// ToString - for tests
func (a MetaData) ToString() string {
	showDataType := true
	keys := make([]string, 0, len(a))
	for k := range a {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := "["
	prefix := ""

	for _, k := range keys {
		result += fmt.Sprintf("%v%v:", prefix, k)

		dt := ""
		switch a[k].DataType {
		case protos.Experiment_MT_STRING:
			result += fmt.Sprintf("%v", a[k].SValue)
			dt = "s"
		case protos.Experiment_MT_INT:
			result += fmt.Sprintf("%v", a[k].IValue)
			dt = "i"
		case protos.Experiment_MT_FLOAT:
			result += fmt.Sprintf("%v", a[k].FValue)
			dt = "f"
		}

		if showDataType {
			result += fmt.Sprintf("/%v", dt)
		}
		prefix = " "
	}
	result += "]"
	return result
}

// DetectorSample - Represents an individual spectrum
type DetectorSample struct {
	// Metadata for the spectrum, string->string
	Meta MetaData
	// Spectrum histogram values, generally 4096 of them
	Spectrum []int64
}

// ToString - for tests
func (d DetectorSample) ToString() string {
	return "meta " + d.Meta.ToString() + " spectrum " + fmt.Sprintf("%+v", d.Spectrum)
}

func (d DetectorSample) detectorID() string {
	return d.Meta["DETECTOR_ID"].SValue // we know it's a string...
}

// DetectorSampleByPMC - Detector data in a lookup by PMC
type DetectorSampleByPMC map[int32][]DetectorSample

// PseudoIntensityRange - Range item
type PseudoIntensityRange struct {
	Name  string
	Start int
	End   int
}

// PseudoIntensities - PMC to pseudo-intensity float array
type PseudoIntensities map[int32][]float32

// BeamLocationProj - Beam location projected onto context image i/j direction
type BeamLocationProj struct {
	I float32
	J float32
}

// BeamLocation - One beam location record for a given PMC. IJ is an array of coordinates
// for each context image (indexed by the PMC of the context image)
type BeamLocation struct {
	X        float32
	Y        float32
	Z        float32
	GeomCorr float32
	IJ       map[int32]BeamLocationProj
}

// BeamLocationByPMC - All beam location info we have per PMC
type BeamLocationByPMC map[int32]BeamLocation

// HousekeepingData - stores column names & data
type HousekeepingData struct {
	Header []string
	Data   map[int32][]MetaValue
}

// PMCData - Used to pass everything we've read to the output saver package...
type PMCData struct {
	Housekeeping      []MetaValue
	Beam              *BeamLocation
	DetectorSpectra   []DetectorSample
	ContextImageSrc   string
	ContextImageDst   string
	PseudoIntensities []float32
}

// FileMetaData - dataset metadata
type FileMetaData struct {
	RTT      int32
	SCLK     int32
	SOL      string
	SiteID   int32
	Site     string
	DriveID  int32
	TargetID string
	Target   string
	Title    string
}

// ImageMeta - metadata for the "disco" image
type ImageMeta struct {
	LEDs     string // R, G, B, U, W (all LEDs on), PC = the processed floating point TIF image
	PMC      int32
	FileName string
	ProdType string
}

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

// OutputData - the outer structure holding everything required to save a dataset
type OutputData struct {
	// dataset ID that should be used to identify this dataset as part of the output path/file name, etc
	DatasetID string
	// The group the dataset will belong to
	Group string

	Meta                FileMetaData
	DetectorConfig      string
	BulkQuantFile       string
	DefaultContextImage string

	// Pseudo-intensity ranges defined for this experiment (they may not change, and this may be redundant!)
	PseudoRanges []PseudoIntensityRange

	// Housekeeping header names
	HousekeepingHeaders []string

	// Per-PMC data
	PerPMCData map[int32]*PMCData

	// RGBU images
	RGBUImages []ImageMeta

	// Disco images - visual spectroscopy using MCC, taken different coloured LEDs
	DISCOImages []ImageMeta

	// Images that reference and match aligned images
	MatchedAlignedImages []MatchedAlignedImageMeta
}

// EnsurePMC - allocates an item to store data for the given PMC if doesn't already exist
func (o *OutputData) EnsurePMC(pmc int32) {
	_, ok := o.PerPMCData[pmc]
	if !ok {
		o.PerPMCData[pmc] = &PMCData{
			Housekeeping:      []MetaValue{},
			Beam:              nil,
			DetectorSpectra:   []DetectorSample{},
			ContextImageSrc:   "",
			ContextImageDst:   "",
			PseudoIntensities: []float32{},
		}
	}
}

// SetPMCData - Passing in all data by PMC lookups
func (o *OutputData) SetPMCData(
	beams BeamLocationByPMC,
	hk HousekeepingData,
	spectra DetectorSampleByPMC,
	contextImgsPerPMC map[int32]string,
	pseudoIntensityData PseudoIntensities) {

	for pmc, beam := range beams {
		o.EnsurePMC(pmc)

		beamCopy := &BeamLocation{
			X:        beam.X,
			Y:        beam.Y,
			Z:        beam.Z,
			GeomCorr: beam.GeomCorr,
			IJ:       map[int32]BeamLocationProj{},
		}

		for beamPMC, ij := range beam.IJ {
			beamCopy.IJ[beamPMC] = BeamLocationProj{I: ij.I, J: ij.J}
		}

		o.PerPMCData[pmc].Beam = beamCopy
	}

	o.HousekeepingHeaders = hk.Header
	for pmc, hkMetaValues := range hk.Data {
		o.EnsurePMC(pmc)
		o.PerPMCData[pmc].Housekeeping = hkMetaValues
	}

	for pmc, detSpectra := range spectra {
		o.EnsurePMC(pmc)
		o.PerPMCData[pmc].DetectorSpectra = detSpectra
	}

	// Find the lowest PMC that has a context image set too
	lowestCtxImgPMC := int32(0)
	for pmc, img := range contextImgsPerPMC {
		o.EnsurePMC(pmc)
		o.PerPMCData[pmc].ContextImageSrc = img

		// Destination image will be just PMC.ext because we don't want to
		// upload the long name, potentially exposing internal meta data
		ext := strings.ToLower(filepath.Ext(img))
		if ext == ".tif" {
			ext = ".png"
		}

		// We used to rename MCC images, now we preserve the name so images can be cross-referenced with MarsViewer/datadrive, etc
		//imgDst := fmt.Sprintf("MCC-%v%v", pmc, ext)
		imgDst := filepath.Base(img)
		imgDst = imgDst[0:len(imgDst)-len(ext)] + ext

		o.PerPMCData[pmc].ContextImageDst = imgDst

		if lowestCtxImgPMC == 0 || pmc < lowestCtxImgPMC {
			lowestCtxImgPMC = pmc

			// Now we can set the default context image file name to the one the lowest PMC points to
			o.DefaultContextImage = imgDst
		}
	}

	for pmc, ps := range pseudoIntensityData {
		o.EnsurePMC(pmc)
		o.PerPMCData[pmc].PseudoIntensities = ps
	}
}
