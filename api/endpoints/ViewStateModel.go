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
	"sort"
	"strings"

	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/utils"
)

// All the structures in a view state config, and their "make" functions:

type spectrumXRFLineState struct {
	LineInfo elementLines `json:"line_info"`
	Visible  bool         `json:"visible"`
}

type energyCalibration struct {
	Detector     string  `json:"detector"`
	EVStart      float32 `json:"eVStart"`
	EVPerChannel float32 `json:"eVPerChannel"`
}

type spectrumLines struct {
	RoiID           string   `json:"roiID"`           // can be "dataset" or "selection" for those special cases
	LineExpressions []string `json:"lineExpressions"` // max(A), bulk(A) or things like sum(max(A), max(B))
}

type spectrumWidgetState struct {
	PanX              float32                `json:"panX"`
	PanY              float32                `json:"panY"`
	ZoomX             float32                `json:"zoomX"`
	ZoomY             float32                `json:"zoomY"`
	SpectrumLines     []spectrumLines        `json:"spectrumLines"`
	LogScale          bool                   `json:"logScale"`
	XRFLines          []spectrumXRFLineState `json:"xrflines"`
	ShowXAsEnergy     bool                   `json:"showXAsEnergy"`
	EnergyCalibration []energyCalibration    `json:"energyCalibration"`
}

type histogramState struct {
	ShowStdDeviation bool     `json:"showStdDeviation"`
	LogScale         bool     `json:"logScale"`
	ExpressionIDs    []string `json:"expressionIDs"`
	VisibleROIs      []string `json:"visibleROIs"`
}

type quantificationState struct {
	AppliedQuantID string `json:"appliedQuantID"`

	// DEPRECATED: This was supposed to replace the above, but went unused. We now load this map and write the first quant ID
	// to the above
	QuantificationByROI map[string]string `json:"quantificationByROI,omitempty"`
}

type selectionState struct {
	// PMC selection world
	SelectedROIID   string `json:"roiID"`
	SelectedROIName string `json:"roiName"`
	SelectedLocIdxs []int  `json:"locIdxs"`

	// PIXEL selection world (Added for RGBU)
	PixelSelectionImageName string `json:"pixelSelectionImageName,omitempty"`
	PixelIdxs               []int  `json:"pixelIdxs,omitempty"`
	CropPixelIdxs           []int  `json:"cropPixelIdxs,omitempty"`
}

type chordState struct {
	ShowForSelection bool     `json:"showForSelection"`
	ExpressionIDs    []string `json:"expressionIDs"`
	DisplayROI       string   `json:"displayROI"`
	Threshold        float32  `json:"threshold"`
	DrawMode         string   `json:"drawMode"`
}

type binaryState struct {
	ShowMmol      bool     `json:"showMmol"`
	ExpressionIDs []string `json:"expressionIDs"`
	VisibleROIs   []string `json:"visibleROIs"`
}

type ternaryState struct {
	ShowMmol      bool     `json:"showMmol"`
	ExpressionIDs []string `json:"expressionIDs"`
	VisibleROIs   []string `json:"visibleROIs"`
}

type tableState struct {
	ShowPureElements bool     `json:"showPureElements"`
	Order            string   `json:"order"`
	VisibleROIs      []string `json:"visibleROIs"`
}

type roiQuantTableState struct {
	ROI      string   `json:"roi"`
	QuantIDs []string `json:"quantIDs"`
}

type variogramState struct {
	ExpressionIDs  []string `json:"expressionIDs"`
	VisibleROIs    []string `json:"visibleROIs"`
	VarioModel     string   `json:"varioModel"` // valid: "exponential", "spherical", "gaussian"
	MaxDistance    float32  `json:"maxDistance"`
	BinCount       int32    `json:"binCount"`
	DrawModeVector bool     `json:"drawModeVector"` // vector or isotropic
}

type mapLayerVisibility struct {
	ExpressionID string  `json:"expressionID"`
	Opacity      float32 `json:"opacity"`
	Visible      bool    `json:"visible"`

	DisplayValueRangeMin float32 `json:"displayValueRangeMin"`
	DisplayValueRangeMax float32 `json:"displayValueRangeMax"`
	DisplayValueShading  string  `json:"displayValueShading"`
}

type roiLayerVisibility struct {
	RoiID   string  `json:"roiID"`
	Opacity float32 `json:"opacity"`
	Visible bool    `json:"visible"`
}

type contextImageState struct {
	PanX                          float32              `json:"panX"`
	PanY                          float32              `json:"panY"`
	ZoomX                         float32              `json:"zoomX"`
	ZoomY                         float32              `json:"zoomY"`
	ShowPoints                    bool                 `json:"showPoints"`
	ShowPointBBox                 bool                 `json:"showPointBBox"`
	PointColourScheme             string               `json:"pointColourScheme"`
	PointBBoxColourScheme         string               `json:"pointBBoxColourScheme"`
	ContextImage                  string               `json:"contextImage"`
	ContextImageSmoothing         string               `json:"contextImageSmoothing"`
	MapLayers                     []mapLayerVisibility `json:"mapLayers"`
	ROILayers                     []roiLayerVisibility `json:"roiLayers"`
	ElementRelativeShading        bool                 `json:"elementRelativeShading"`
	Brightness                    float32              `json:"brightness"`
	RGBUChannels                  string               `json:"rgbuChannels"`
	UnselectedOpacity             float32              `json:"unselectedOpacity"`
	UnselectedGrayscale           bool                 `json:"unselectedGrayscale"`
	ColourRatioMin                float32              `json:"colourRatioMin"`
	ColourRatioMax                float32              `json:"colourRatioMax"`
	RemoveTopSpecularArtifacts    bool                 `json:"removeTopSpecularArtifacts"`
	RemoveBottomSpecularArtifacts bool                 `json:"removeBottomSpecularArtifacts"`

	// Could store per-tool state
	//ActiveTool string `json:"activeTool"`
	// SelectionAdditiveMode bool `json:"selectionAdditiveMode"`
	// PMCToolPMC int32 `json:"pmcToolPMC"`
}

type AnnotationPoint struct {
	X            float32 `json:"x"`
	Y            float32 `json:"y"`
	ScreenWidth  float32 `json:"screenWidth"`
	ScreenHeight float32 `json:"screenHeight"`
}

type FullScreenAnnotationItem struct {
	Type     string            `json:"type"`
	Points   []AnnotationPoint `json:"points"`
	Colour   string            `json:"colour"`
	Complete bool              `json:"complete"`
	Text     string            `json:"text,omitempty"`
	FontSize int               `json:"fontSize,omitempty"`
	ID       int               `json:"id,omitempty"`
}

type annotationDisplayState struct {
	SavedAnnotations []FullScreenAnnotationItem `json:"savedAnnotations"`
}

type roiDisplayState struct {
	ROIColours map[string]string `json:"roiColours"`
	ROIShapes  map[string]string `json:"roiShapes"`
}

type rgbuPlotWidgetState struct {
	Minerals          []string `json:"minerals"`
	YChannelA         string   `json:"yChannelA"`
	YChannelB         string   `json:"yChannelB"`
	XChannelA         string   `json:"xChannelA"`
	XChannelB         string   `json:"xChannelB"`
	DrawMonochrome    bool     `json:"drawMonochrome"`
	SelectedMinXValue float32  `json:"selectedMinXValue"`
	SelectedMaxXValue float32  `json:"selectedMaxXValue"`
	SelectedMinYValue float32  `json:"selectedMinYValue"`
	SelectedMaxYValue float32  `json:"selectedMaxYValue"`
}

type singleAxisRGBUWidgetState struct {
	Minerals          []string `json:"minerals"`
	ChannelA          string   `json:"channelA"`
	ChannelB          string   `json:"channelB"`
	RoiStackedOverlap bool     `json:"roiStackedOverlap"`
}

type rgbuImagesWidgetState struct {
	LogColour  bool    `json:"logColour"`
	Brightness float32 `json:"brightness"`
}

type parallelogramWidgetState struct {
	Regions  []string `json:"regions"`
	Channels []string `json:"channels"`
}

// Any state saved for the overall analysis tab
type analysisLayoutState struct {
	TopWidgetSelectors    []string `json:"topWidgetSelectors"`
	BottomWidgetSelectors []string `json:"bottomWidgetSelectors"`
}

// Some of these have single copies, because it only makes sense to have one of them, while others have a map lookup
// where the name of the specific config is stored as a lookup. This is because for example, the user can configure multiple
type wholeViewState struct {
	AnalysisLayout analysisLayoutState `json:"analysisLayout"`

	// Deprecated - we used to enforce only one spectrum in the top-right. Now not restricted, so if an old state file
	// is loaded, we still allow for this, but new state files should be written with this in the spectrums field
	Spectrum spectrumWidgetState `json:"spectrum"`

	ContextImages  map[string]contextImageState         `json:"contextImages"`
	Histograms     map[string]histogramState            `json:"histograms"`
	ChordDiagrams  map[string]chordState                `json:"chordDiagrams"`
	TernaryPlots   map[string]ternaryState              `json:"ternaryPlots"`
	BinaryPlots    map[string]binaryState               `json:"binaryPlots"`
	Tables         map[string]tableState                `json:"tables"`
	ROIQuantTables map[string]roiQuantTableState        `json:"roiQuantTables"`
	Variograms     map[string]variogramState            `json:"variograms"`
	Spectrums      map[string]spectrumWidgetState       `json:"spectrums"`
	RGBUPlots      map[string]rgbuPlotWidgetState       `json:"rgbuPlots"`
	SingleAxisRGBU map[string]singleAxisRGBUWidgetState `json:"singleAxisRGBU"`
	RGBUImageViews map[string]rgbuImagesWidgetState     `json:"rgbuImages"`
	Parallelograms map[string]parallelogramWidgetState  `json:"parallelograms"`

	// Full Screen Annotations
	Annotations annotationDisplayState `json:"annotations"`

	// Not strictly the view-state of a widget, but the shared display state of ROIs
	// for the given user/dataset
	ROIs roiDisplayState `json:"rois"`

	// Loaded quantification ID
	Quantification quantificationState `json:"quantification"`

	// Again not the state of a single widget but the shared selection all widgets can show
	Selection selectionState `json:"selection"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Defaults for widgets

func defaultContextImage() contextImageState {
	img := contextImageState{}
	img.ZoomX = 1
	img.ZoomY = 1
	img.ContextImageSmoothing = "linear"
	img.Brightness = 1
	img.RGBUChannels = "RGB"
	img.UnselectedOpacity = 0.4
	img.UnselectedGrayscale = false
	img.MapLayers = []mapLayerVisibility{}
	img.ROILayers = []roiLayerVisibility{}
	img.ShowPoints = true
	img.ShowPointBBox = true
	img.PointColourScheme = "PURPLE_CYAN"
	img.PointBBoxColourScheme = "PURPLE_CYAN"
	img.ElementRelativeShading = true
	return img
}

func defaultChordDiagram() chordState {
	chord := chordState{}
	chord.ExpressionIDs = []string{}
	chord.DrawMode = "BOTH"
	return chord
}

func defaultBinaryPlot() binaryState {
	binary := binaryState{}
	binary.ExpressionIDs = []string{}
	binary.VisibleROIs = []string{}
	return binary
}

func defaultTernaryPlot() ternaryState {
	ternary := ternaryState{}
	ternary.ExpressionIDs = []string{}
	ternary.VisibleROIs = []string{}
	return ternary
}

func defaultTable() tableState {
	table := tableState{}
	table.ShowPureElements = false
	table.Order = "atomic-number"
	table.VisibleROIs = []string{}
	return table
}

func defaultROIQuantTable() roiQuantTableState {
	table := roiQuantTableState{}
	table.ROI = ""
	table.QuantIDs = []string{}
	return table
}

func defaultHistogram() histogramState {
	hist := histogramState{}
	hist.ExpressionIDs = []string{}
	hist.VisibleROIs = []string{}
	return hist
}

func defaultVariogram() variogramState {
	vario := variogramState{}
	vario.ExpressionIDs = []string{}
	vario.VisibleROIs = []string{}
	vario.DrawModeVector = false
	vario.VarioModel = "exponential"
	return vario
}

func defaultRGBUPlot() rgbuPlotWidgetState {
	rgbu := rgbuPlotWidgetState{}
	rgbu.Minerals = []string{}
	return rgbu
}

func defaultSingleAxisRGBU() singleAxisRGBUWidgetState {
	singleAxisRGBU := singleAxisRGBUWidgetState{}
	singleAxisRGBU.Minerals = []string{}
	return singleAxisRGBU
}

func defaultRGBUImages() rgbuImagesWidgetState {
	rgbu := rgbuImagesWidgetState{}
	return rgbu
}

func defaultParallelogram() parallelogramWidgetState {
	parallelogram := parallelogramWidgetState{}
	parallelogram.Regions = []string{}
	parallelogram.Channels = []string{}
	return parallelogram
}

func defaultSelectionState() selectionState {
	var state selectionState
	state.SelectedLocIdxs = []int{}
	state.PixelIdxs = []int{}
	state.CropPixelIdxs = []int{}
	return state
}

func defaultSpectrum() spectrumWidgetState {
	var state spectrumWidgetState
	state.ZoomX = 1
	state.ZoomY = 1
	state.LogScale = true
	state.SpectrumLines = []spectrumLines{}
	state.XRFLines = []spectrumXRFLineState{}
	state.ShowXAsEnergy = false
	state.EnergyCalibration = []energyCalibration{}
	return state
}

func defaultWholeViewState() wholeViewState {
	// Set up a default view state
	state := wholeViewState{}

	// Don't supply invalid defaults!
	state.AnalysisLayout.TopWidgetSelectors = []string{}
	state.AnalysisLayout.BottomWidgetSelectors = []string{}

	state.Spectrum = defaultSpectrum()

	state.Selection = defaultSelectionState()

	state.Annotations = annotationDisplayState{}
	state.Annotations.SavedAnnotations = []FullScreenAnnotationItem{}

	state.ROIs = roiDisplayState{}
	state.ROIs.ROIColours = map[string]string{}
	state.ROIs.ROIShapes = map[string]string{}

	state.Histograms = map[string]histogramState{}
	state.ChordDiagrams = map[string]chordState{}
	state.TernaryPlots = map[string]ternaryState{}
	state.Tables = map[string]tableState{}
	state.ROIQuantTables = map[string]roiQuantTableState{}
	state.BinaryPlots = map[string]binaryState{}
	state.Variograms = map[string]variogramState{}
	state.ContextImages = map[string]contextImageState{}
	state.Spectrums = map[string]spectrumWidgetState{}
	state.RGBUImageViews = map[string]rgbuImagesWidgetState{}
	state.RGBUPlots = map[string]rgbuPlotWidgetState{}
	state.SingleAxisRGBU = map[string]singleAxisRGBUWidgetState{}
	state.Parallelograms = map[string]parallelogramWidgetState{}

	state.Quantification.QuantificationByROI = map[string]string{}

	return state
}

// Other setup functions

func applyQuantByROIFallback(quantState *quantificationState) {
	// NOTE: we had per-ROI quants for a while but removed this because this was the first attempt at supporting multi-quant
	// but we have a whole bunch of view states saved that have this map. Here we read the quant out and put it in the original single field
	if len(quantState.QuantificationByROI) > 0 && len(quantState.AppliedQuantID) <= 0 {

		// This is a bit slower, but again due to unit tests not being deterministic, we need to sort
		// so here we collect all non-AllPoints Quants we find...
		quantIDs := []string{}

		for roiID, quantID := range quantState.QuantificationByROI {
			if roiID == "AllPoints" {
				quantState.AppliedQuantID = quantID
				break // Found AllPoints, this is the "authoritative" one
			} else if len(quantID) > 0 {
				quantIDs = append(quantIDs, quantID)
			}
		}

		// If we still haven't found the AllPoints quant, apply anything else we may have seen
		if len(quantState.AppliedQuantID) <= 0 && len(quantIDs) > 0 {
			// Sort the list alphabetically
			sort.Strings(quantIDs)
			quantState.AppliedQuantID = quantIDs[0]
		}
	}

	// Clear the unused field, this means it shouldn't appear in JSON
	quantState.QuantificationByROI = map[string]string{}
}

// Getting all IDs referenced by a view state
type referencedIDItem struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Creator pixlUser.UserInfo `json:"creator"`
}

type viewStateReferencedIDs struct {
	Quant       referencedIDItem   `json:"quant,omitempty"`
	ROIs        []referencedIDItem `json:"ROIs"`
	Expressions []referencedIDItem `json:"expressions"`
	RGBMixes    []referencedIDItem `json:"rgbMixes"`

	NonSharedCount int `json:"nonSharedCount"`
}

/* Items checked... hope this is all!

Histograms[].VisibleROIs[]
Histograms[].ExpressionIDs[]
Spectrums[].SpectrumLines[].RoiID
ContextImages[].ROILayers[].RoiID
ContextImages[].MapLayers[].ExpressionID
Variograms[].VisibleROIs[]
Variograms[].ExpressionIDs[]
ROIQuantTables[].ROI
Tables[].VisibleROIs[]
TernaryPlots[].VisibleROIs[]
TernaryPlots[].ExpressionIDs[]
BinaryPlots[].VisibleROIs[]
BinaryPlots[].ExpressionIDs[]
ChordDiagrams[].DisplayROI
ChordDiagrams[].ExpressionIDs[]
ROIs.ROIColours keys <-- don't care, these are just ids

ROIQuantTables[].QuantIDs[] <-- don't care, too complicated for now
Quantification.AppliedQuantID
*/

func (state wholeViewState) getReferencedIDs() viewStateReferencedIDs {
	// Unfortunately this has to be manually coded... can't exactly search for field names or something to identify them

	// We use maps here for uniqueness
	roiIDs := map[string]bool{}
	expressionIDs := map[string]bool{}

	// Scan for ROI IDs and Expression IDs
	for _, item := range state.Histograms {
		utils.SetStringsInMap(item.VisibleROIs, roiIDs)
		utils.SetStringsInMap(item.ExpressionIDs, expressionIDs)
	}

	// Not needed?
	for _, item := range state.Spectrums {
		for _, line := range item.SpectrumLines {
			roiIDs[line.RoiID] = true
		}
	}

	for _, item := range state.ContextImages {
		for _, layer := range item.ROILayers {
			roiIDs[layer.RoiID] = true
		}
		for _, layer := range item.MapLayers {
			expressionIDs[layer.ExpressionID] = true
		}
	}

	for _, item := range state.Variograms {
		utils.SetStringsInMap(item.VisibleROIs, roiIDs)
		utils.SetStringsInMap(item.ExpressionIDs, expressionIDs)
	}

	for _, item := range state.ROIQuantTables {
		roiIDs[item.ROI] = true
		// Too hard... result.ROIIDs = append(result.QuantID, item.QuantIDs...)
	}

	for _, item := range state.Tables {
		utils.SetStringsInMap(item.VisibleROIs, roiIDs)
	}

	for _, item := range state.TernaryPlots {
		utils.SetStringsInMap(item.VisibleROIs, roiIDs)
		utils.SetStringsInMap(item.ExpressionIDs, expressionIDs)
	}

	for _, item := range state.BinaryPlots {
		utils.SetStringsInMap(item.VisibleROIs, roiIDs)
		utils.SetStringsInMap(item.ExpressionIDs, expressionIDs)
	}

	for _, item := range state.ChordDiagrams {
		roiIDs[item.DisplayROI] = true
		utils.SetStringsInMap(item.ExpressionIDs, expressionIDs)
	}

	result := viewStateReferencedIDs{
		ROIs:        []referencedIDItem{},
		Expressions: []referencedIDItem{},
		RGBMixes:    []referencedIDItem{},
	}

	// NOTE: there's a whole bunch of stuff we want to filter out, so do that here!
	for _, roiID := range utils.GetStringMapKeys(roiIDs) {
		// Ignore predefined ROIs:
		if roiID != "SelectedPoints" && roiID != "AllPoints" && roiID != "Remaining Points" {
			result.ROIs = append(result.ROIs, referencedIDItem{
				ID: roiID,
			})
		}
	}

	for _, exprID := range utils.GetStringMapKeys(expressionIDs) {
		// We don't care about predefined expressions on the UI, but do care about
		// user-defined expressions and RGB mixes
		if strings.HasPrefix(exprID, "rgbmix-") || strings.HasPrefix(exprID, "shared-rgbmix-") {
			result.RGBMixes = append(result.RGBMixes, referencedIDItem{
				ID: exprID,
			})
		} else if !strings.HasPrefix(exprID, "expr-") {
			result.Expressions = append(result.Expressions, referencedIDItem{
				ID: exprID,
			})
		}
	}

	// This one's easy
	result.Quant = referencedIDItem{
		ID: state.Quantification.AppliedQuantID,
	}

	// Count how many are not shared ids
	for _, item := range result.ROIs {
		if !strings.HasPrefix(item.ID, utils.SharedItemIDPrefix) {
			result.NonSharedCount++
		}
	}
	for _, item := range result.Expressions {
		if !strings.HasPrefix(item.ID, utils.SharedItemIDPrefix) {
			result.NonSharedCount++
		}
	}
	if !strings.HasPrefix(result.Quant.ID, utils.SharedItemIDPrefix) {
		result.NonSharedCount++
	}

	return result
}

func (state wholeViewState) replaceReferencedIDs(replacements map[string]string) {
	// Scan for ROI IDs and Expression IDs
	for _, item := range state.Histograms {
		utils.ReplaceStringsInSlice(item.VisibleROIs, replacements)
		utils.ReplaceStringsInSlice(item.ExpressionIDs, replacements)
	}

	// Not needed?
	for _, item := range state.Spectrums {
		for _, line := range item.SpectrumLines {
			if replacement, ok := replacements[line.RoiID]; ok {
				line.RoiID = replacement
			}
		}
	}

	for _, item := range state.ContextImages {
		for _, layer := range item.ROILayers {
			if replacement, ok := replacements[layer.RoiID]; ok {
				layer.RoiID = replacement
			}
		}
		for _, layer := range item.MapLayers {
			if replacement, ok := replacements[layer.ExpressionID]; ok {
				layer.ExpressionID = replacement
			}
		}
	}

	for _, item := range state.Variograms {
		utils.ReplaceStringsInSlice(item.VisibleROIs, replacements)
		utils.ReplaceStringsInSlice(item.ExpressionIDs, replacements)
	}

	for _, item := range state.ROIQuantTables {
		if replacement, ok := replacements[item.ROI]; ok {
			item.ROI = replacement
		}
		// Too hard... result.ROIIDs = append(result.QuantID, item.QuantIDs...)
	}

	for _, item := range state.Tables {
		utils.ReplaceStringsInSlice(item.VisibleROIs, replacements)
	}

	for _, item := range state.TernaryPlots {
		utils.ReplaceStringsInSlice(item.VisibleROIs, replacements)
		utils.ReplaceStringsInSlice(item.ExpressionIDs, replacements)
	}

	for _, item := range state.BinaryPlots {
		utils.ReplaceStringsInSlice(item.VisibleROIs, replacements)
		utils.ReplaceStringsInSlice(item.ExpressionIDs, replacements)
	}

	for _, item := range state.ChordDiagrams {
		if replacement, ok := replacements[item.DisplayROI]; ok {
			item.DisplayROI = replacement
		}
		utils.ReplaceStringsInSlice(item.ExpressionIDs, replacements)
	}

	// This one's easy
	if replacement, ok := replacements[state.Quantification.AppliedQuantID]; ok {
		state.Quantification.AppliedQuantID = replacement
	}
}
