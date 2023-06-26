package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func getViewStateFiles(state *wholeViewState, fs fileaccess.FileAccess, bucket string, datasetID string, userID string) error {
	items, err := fs.ListObjects(
		bucket,
		filepaths.GetViewStatePath(userID, datasetID, "")+"/",
	)
	if err != nil {
		return err
	}

	for _, path := range items {
		if filepath.Ext(path) != ".json" {
			continue // skip, it's not something we care about
		}

		fileNameOnly := filepath.Base(path[0 : len(path)-5])
		dataType, whichInstance := splitWidgetFileName(fileNameOnly)

		// Try read in to ones which only return a single object (not looking for whichInstance)
		if dataType == "spectrum" && len(whichInstance) <= 0 {
			// OLD STYLE spectrum pre configurable spectrum position, only here for backwards compatibility with existing
			// view states newer ones are saved with whichInstance set to something to identify their position
			err := fs.ReadJSON(bucket, path, &state.Spectrum, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}
		} else if dataType == "quantification" {
			err = fs.ReadJSON(bucket, path, &state.Quantification, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}

			applyQuantByROIFallback(&state.Quantification)
		} else if dataType == "selection" {
			err = fs.ReadJSON(bucket, path, &state.Selection, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}
		} else if dataType == "roi" {
			err = fs.ReadJSON(bucket, path, &state.ROIs, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}
		} else if dataType == "analysisLayout" {
			err = fs.ReadJSON(bucket, path, &state.AnalysisLayout, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}
		} else if dataType == "annotations" {
			err = fs.ReadJSON(bucket, path, &state.Annotations, false)
			if err != nil && !fs.IsNotFoundError(err) {
				return err
			}
		} else if len(whichInstance) > 0 {
			// Check the ones where we DO require an instance name
			if dataType == "contextImage" {
				contextImage := defaultContextImage()
				err = fs.ReadJSON(bucket, path, &contextImage, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.ContextImages[whichInstance] = contextImage
			} else if dataType == "histogram" {
				histogram := defaultHistogram()
				err = fs.ReadJSON(bucket, path, &histogram, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.Histograms[whichInstance] = histogram
			} else if dataType == "chord" {
				chord := defaultChordDiagram()
				err = fs.ReadJSON(bucket, path, &chord, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.ChordDiagrams[whichInstance] = chord
			} else if dataType == "ternary" {
				ternary := defaultTernaryPlot()
				err = fs.ReadJSON(bucket, path, &ternary, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.TernaryPlots[whichInstance] = ternary
			} else if dataType == "binary" {
				binary := defaultBinaryPlot()
				err = fs.ReadJSON(bucket, path, &binary, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.BinaryPlots[whichInstance] = binary
			} else if dataType == "table" {
				table := defaultTable()
				err = fs.ReadJSON(bucket, path, &table, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.Tables[whichInstance] = table
			} else if dataType == "roiQuantTable" {
				table := defaultROIQuantTable()
				err = fs.ReadJSON(bucket, path, &table, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.ROIQuantTables[whichInstance] = table
			} else if dataType == "variogram" {
				vario := defaultVariogram()
				err = fs.ReadJSON(bucket, path, &vario, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.Variograms[whichInstance] = vario
			} else if dataType == "rgbuPlot" {
				rgbu := defaultRGBUPlot()
				err = fs.ReadJSON(bucket, path, &rgbu, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.RGBUPlots[whichInstance] = rgbu
			} else if dataType == "singleAxisRGBU" {
				singleAxisRGBU := defaultSingleAxisRGBU()
				err = fs.ReadJSON(bucket, path, &singleAxisRGBU, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.SingleAxisRGBU[whichInstance] = singleAxisRGBU
			} else if dataType == "rgbuImages" {
				rgbu := defaultRGBUImages()
				err = fs.ReadJSON(bucket, path, &rgbu, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.RGBUImageViews[whichInstance] = rgbu
			} else if dataType == "parallelogram" {
				pgram := defaultParallelogram()
				err = fs.ReadJSON(bucket, path, &pgram, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.Parallelograms[whichInstance] = pgram
			} else if dataType == "spectrum" {
				spectrum := defaultSpectrum()
				err := fs.ReadJSON(bucket, path, &spectrum, false)
				if err != nil && !fs.IsNotFoundError(err) {
					return err
				}
				state.Spectrums[whichInstance] = spectrum
			}
		}
	}

	filterUnusedWidgetStates(state)

	return nil
}

func filterUnusedWidgetStates(state *wholeViewState) {
	// Make a list of allowed items from the analysis layout values
	// Unfortunately the names of things don't match in analysis layout vs the file names for the widgets :(
	// This is due to the evolution of our saving of view state, early on we didn't even have configurable layouts
	// and things were added gradually.

	// NOTE: we don't do any filtering unless the layout sections have been filled out. For example, with a blank view
	// state we want the UI to decide what to do with the widget data we pass in
	if len(state.AnalysisLayout.TopWidgetSelectors) != 2 || len(state.AnalysisLayout.BottomWidgetSelectors) != 4 {
		return
	}

	layoutNameToWidgetFileName := map[string]string{
		"chord-view-widget":       "chord",
		"binary-plot-widget":      "binary",
		"ternary-plot-widget":     "ternary",
		"table-widget":            "table",
		"histogram-widget":        "histogram",
		"variogram-widget":        "variogram",
		"rgbu-viewer-widget":      "rgbuImages",
		"rgbu-plot-widget":        "rgbuPlot",
		"single-axis-rgbu-widget": "singleAxisRGBU",
		"parallel-coords-widget":  "parallelogram",
		"spectrum-widget":         "spectrum",
		"context-image":           "contextImage",
		"roi-quant-table-widget":  "roiQuantTable",
	}

	// These are always allowed through...
	allowedItems := []string{"contextImage-analysis", "contextImage-map"}

	for c := 0; c < len(state.AnalysisLayout.TopWidgetSelectors); c++ {
		allowedItems = append(allowedItems, layoutNameToWidgetFileName[state.AnalysisLayout.TopWidgetSelectors[c]]+"-top"+strconv.Itoa(c))
	}

	for c := 0; c < len(state.AnalysisLayout.BottomWidgetSelectors); c++ {
		posName := "undercontext"
		if c > 0 {
			posName = "underspectrum" + strconv.Itoa(c-1)
		}
		allowedItems = append(allowedItems, layoutNameToWidgetFileName[state.AnalysisLayout.BottomWidgetSelectors[c]]+"-"+posName)
	}

	for widgetIdentifier := range state.ContextImages {
		if !utils.StringInSlice("contextImage-"+widgetIdentifier, allowedItems) {
			delete(state.ContextImages, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.Histograms {
		if !utils.StringInSlice("histogram-"+widgetIdentifier, allowedItems) {
			delete(state.Histograms, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.ChordDiagrams {
		if !utils.StringInSlice("chord-"+widgetIdentifier, allowedItems) {
			delete(state.ChordDiagrams, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.BinaryPlots {
		if !utils.StringInSlice("binary-"+widgetIdentifier, allowedItems) {
			delete(state.BinaryPlots, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.TernaryPlots {
		if !utils.StringInSlice("ternary-"+widgetIdentifier, allowedItems) {
			delete(state.TernaryPlots, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.Tables {
		if !utils.StringInSlice("table-"+widgetIdentifier, allowedItems) {
			delete(state.Tables, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.ROIQuantTables {
		if !utils.StringInSlice("roiQuantTable-"+widgetIdentifier, allowedItems) {
			delete(state.ROIQuantTables, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.Variograms {
		if !utils.StringInSlice("variogram-"+widgetIdentifier, allowedItems) {
			delete(state.Variograms, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.RGBUPlots {
		if !utils.StringInSlice("rgbuPlot-"+widgetIdentifier, allowedItems) {
			delete(state.RGBUPlots, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.SingleAxisRGBU {
		if !utils.StringInSlice("singleAxisRGBU-"+widgetIdentifier, allowedItems) {
			delete(state.SingleAxisRGBU, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.RGBUImageViews {
		if !utils.StringInSlice("rgbuImages-"+widgetIdentifier, allowedItems) {
			delete(state.RGBUImageViews, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.Parallelograms {
		if !utils.StringInSlice("parallelogram-"+widgetIdentifier, allowedItems) {
			delete(state.Parallelograms, widgetIdentifier)
		}
	}
	for widgetIdentifier := range state.Spectrums {
		if !utils.StringInSlice("spectrum-"+widgetIdentifier, allowedItems) {
			delete(state.Spectrums, widgetIdentifier)
		}
	}
}

func splitWidgetFileName(fileNameOnly string) (string, string) {
	dataType := fileNameOnly
	whichInstance := ""

	dashIdx := strings.Index(fileNameOnly, "-")
	if dashIdx > 0 {
		dataType = fileNameOnly[0:dashIdx]
		whichInstance = fileNameOnly[dashIdx+1:]
	}

	return dataType, whichInstance
}

func migrateViewStates(userContentBucket string, userContentFiles []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	const collectionName = "viewStates"

	err := dest.Collection(collectionName).Drop(context.TODO())
	if err != nil {
		return err
	}

	destViewStates := []interface{}{}

	viewStatesToSave := map[string][]string{}

	for _, p := range userContentFiles {
		if filepath.Base(filepath.Dir(p)) == "ViewState" {
			if strings.HasPrefix(p, "UserContent/shared/") {
				return fmt.Errorf("Unexpected view state path: %v\n", p)
			} else {
				datasetId := filepath.Base(filepath.Dir(filepath.Dir(p)))
				userId := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(p))))

				// Form an id that should make a scan unique
				id := userId + "_" + datasetId

				viewStatesToSave[id] = []string{userId, datasetId}
			}
		}
	}

	// We've found all the view states we want to save, now do those, because each view state
	// spans multiple files
	for id, bits := range viewStatesToSave {
		if maxItemsToRead > 0 && len(destViewStates) >= maxItemsToRead {
			break
		}

		userId := bits[0]
		datasetId := bits[1]

		state := defaultWholeViewState()
		err = getViewStateFiles(&state, fs, userContentBucket, datasetId, userId)
		if err != nil {
			return err
		}

		contextImages := map[string]*protos.ContextImageState{}
		histograms := map[string]*protos.HistogramState{}
		chordDiagrams := map[string]*protos.ChordState{}
		ternaryPlots := map[string]*protos.TernaryState{}
		binaryPlots := map[string]*protos.BinaryState{}
		tables := map[string]*protos.TableState{}
		roiQuantTables := map[string]*protos.ROIQuantTableState{}
		variograms := map[string]*protos.VariogramState{}
		spectrums := map[string]*protos.SpectrumWidgetState{}
		rgbuPlots := map[string]*protos.RGBUPlotWidgetState{}
		singleAxisRGBU := map[string]*protos.SingleAxisRGBUWidgetState{}
		rgbuImages := map[string]*protos.RGBUImagesWidgetState{}
		parallelograms := map[string]*protos.ParallelogramWidgetState{}

		destState := protos.ViewState{
			Id:     id,
			ScanId: datasetId,
			UserId: userId,
			AnalysisLayout: &protos.AnalysisLayout{
				TopWidgetSelectors:    state.AnalysisLayout.TopWidgetSelectors,
				BottomWidgetSelectors: state.AnalysisLayout.BottomWidgetSelectors,
			},
			ContextImages:  contextImages,
			Histograms:     histograms,
			ChordDiagrams:  chordDiagrams,
			TernaryPlots:   ternaryPlots,
			BinaryPlots:    binaryPlots,
			Tables:         tables,
			RoiQuantTables: roiQuantTables,
			Variograms:     variograms,
			Spectrums:      spectrums,
			RgbuPlots:      rgbuPlots,
			SingleAxisRGBU: singleAxisRGBU,
			RgbuImages:     rgbuImages,
			Parallelograms: parallelograms,
			Annotations: &protos.AnnotationDisplayState{
				SavedAnnotations: makeAnnotationStates(state.Annotations.SavedAnnotations),
			},
			Rois: &protos.ROIDisplayState{
				RoiColours: state.ROIs.ROIColours,
				RoiShapes:  state.ROIs.ROIShapes,
			},
			Quantification: &protos.QuantificationState{
				AppliedQuantID: state.Quantification.AppliedQuantID,
			},
			Selection: &protos.SelectionState{
				RoiID:                   state.Selection.SelectedROIID,
				RoiName:                 state.Selection.SelectedROIName,
				LocIdxs:                 toInt32s(state.Selection.SelectedLocIdxs),
				PixelSelectionImageName: state.Selection.PixelSelectionImageName,
				PixelIdxs:               toInt32s(state.Selection.PixelIdxs),
				CropPixelIdxs:           toInt32s(state.Selection.CropPixelIdxs),
			},
		}

		destViewStates = append(destViewStates, destState)

		if len(destViewStates)%10 == 0 {
			fmt.Printf("  Read %v%% (%v of %v) view states...\n", len(destViewStates)*100/len(viewStatesToSave), len(destViewStates), len(viewStatesToSave))
		}
	}

	result, err := dest.Collection(collectionName).InsertMany(context.TODO(), destViewStates)
	if err != nil {
		return err
	}

	fmt.Printf("View states inserted: %v\n", len(result.InsertedIDs))

	return err
}

func toInt32s(a []int) []int32 {
	result := []int32{}
	for _, i := range a {
		result = append(result, int32(i))
	}
	return result
}

func makeAnnotationStates(items []FullScreenAnnotationItem) []*protos.FullScreenAnnotationItem {
	result := []*protos.FullScreenAnnotationItem{}
	for _, item := range items {
		pts := []*protos.AnnotationPoint{}
		for _, pt := range item.Points {
			pts = append(pts, &protos.AnnotationPoint{
				X:            pt.X,
				Y:            pt.Y,
				ScreenHeight: pt.ScreenHeight,
				ScreenWidth:  pt.ScreenWidth,
			})
		}
		result = append(result, &protos.FullScreenAnnotationItem{
			Type:     item.Type,
			Points:   pts,
			Colour:   item.Colour,
			Complete: item.Complete,
			Text:     item.Text,
			FontSize: int32(item.FontSize),
			Id:       int32(item.ID),
		})
	}
	return result
}
