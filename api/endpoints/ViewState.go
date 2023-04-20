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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/quantModel"
	"github.com/pixlise/core/v3/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ViewState - What the user had loaded/set up last, eg quantifications, ROIs, annotations, expressions
// This just references by name/id/whatever the items in question

func registerViewStateHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "view-state"
	const savedURIPath = "/saved"
	const collectionURIPath = "/collections"

	// "Current" view state, as saved by widgets as we go along, and the GET call to retrieve it
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), viewStateList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWritePIXLISESettings), viewStatePut)

	// Saved view states - these are named copies of a view state, with CRUD calls
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), savedViewStateGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWritePIXLISESettings), savedViewStatePut)
	// Renaming a view state
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier)+"/references", apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), savedViewStateGetReferencedIDs)
	//router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier)+"/rename", apiRouter.MakeMethodPermission("POST", permission.PermWritePIXLISESettings), savedViewStateRenamePost)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWritePIXLISESettings), savedViewStateDelete)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), savedViewStateList)

	// Collections (of saved view states)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+collectionURIPath, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), viewStateCollectionList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+collectionURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPIXLISESettings), viewStateCollectionGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+collectionURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWritePIXLISESettings), viewStateCollectionPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+collectionURIPath, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWritePIXLISESettings), viewStateCollectionDelete)

	// Sharing the above
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWritePIXLISESettings), viewStateShare)
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix+"-collection", datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWritePIXLISESettings), viewStateCollectionShare)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// "Current" View State

func viewStateList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	// It's a get, we don't care about the body...

	state := defaultWholeViewState()

	// If user is NOT resetting, overwrite what we just made with the stored view state. If anything is not in there, it retains the default state
	if resetVal, ok := params.PathParams["reset"]; ok && resetVal == "true" {
		// Clear all view state files
		err := clearLastViewStateFiles(params)
		if err != nil {
			return nil, err
		}
	}

	// Load view state files (if any)
	getViewStateFiles(&state, params.Svcs.FS, params.Svcs.Config.UsersBucket, datasetID, params.UserInfo.UserID)

	// If at the end of all this, the quantification state applied is still empty, check if there is a "blessed" quantification that
	// we could auto-load
	if len(state.Quantification.AppliedQuantID) <= 0 {
		_, blessItem, _, err := quantModel.GetBlessedQuantFile(params.Svcs, datasetID)
		if err == nil && blessItem != nil && len(blessItem.JobID) > 0 {
			quantId := utils.SharedItemIDPrefix + blessItem.JobID

			state.Quantification.AppliedQuantID = quantId
		}
	}

	return &state, nil
}

func clearLastViewStateFiles(params handlers.ApiHandlerParams) error {
	datasetID := params.PathParams[datasetIdentifier]

	// List all files in the path
	listing, err := params.Svcs.FS.ListObjects(
		params.Svcs.Config.UsersBucket,
		filepaths.GetViewStatePath(params.UserInfo.UserID, datasetID, "")+"/",
	)
	if err != nil {
		return err
	}

	// Delete them all (ensure they don't contain workspace/collection files, check within the path)
	fails := []string{}
	for _, item := range listing {
		if !strings.Contains(item, "/"+filepaths.ViewStateSavedSubpath+"/") &&
			!strings.Contains(item, "/"+filepaths.ViewStateCollectionsSubpath+"/") {
			err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, item)
			if err != nil {
				fails = append(fails, item)
			}
		}
	}

	// If we had errors, report them back
	if len(fails) > 0 {
		return fmt.Errorf("Failed to delete files: %v", strings.Join(fails, ","))
	}

	return nil
}

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

func viewStatePut(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	identifier := params.PathParams[idIdentifier]

	// If it's all, we just got given an entire view state in one message, so we save
	// it differently.
	if identifier == "all" {
		return saveAllState(params, body)
	}

	// Get the widget type/instance
	dataType, whichInstance := splitWidgetFileName(identifier)

	// For every widget, we have a separate save method
	// First try saving the ones that are singular
	if len(whichInstance) <= 0 {
		switch dataType {
		case "annotations":
			return saveAnnotationState(params, body)
		case "roi":
			return saveROIState(params, body)
		case "quantification":
			return saveQuantificationState(params, body)
		case "selection":
			return saveSelectionState(params, body)
		case "analysisLayout":
			return saveAnalysisLayoutState(params, body)
		}
	} else {
		switch dataType {
		case "contextImage":
			return saveContextImageState(params, body, whichInstance)
		case "histogram":
			return saveHistogramState(params, body, whichInstance)
		case "chord":
			return saveChordState(params, body, whichInstance)
		case "binary":
			return saveBinaryState(params, body, whichInstance)
		case "ternary":
			return saveTernaryState(params, body, whichInstance)
		case "table":
			return saveTableState(params, body, whichInstance)
		case "roiQuantTable":
			return saveROIQuantTableState(params, body, whichInstance)
		case "variogram":
			return saveVariogramState(params, body, whichInstance)
		case "rgbuPlot":
			return saveRGBUPlotState(params, body, whichInstance)
		case "singleAxisRGBU":
			return saveSingleAxisRGBUState(params, body, whichInstance)
		case "rgbuImages":
			return saveRGBUImagesState(params, body, whichInstance)
		case "parallelogram":
			return saveParallelogramState(params, body, whichInstance)
		case "spectrum":
			return saveSpectrumState(params, body, whichInstance)
		}
	}

	return nil, api.MakeBadRequestError(fmt.Errorf("Unknown widget: %v", dataType))
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Saving individual view state files in "current" state. This is done this way because each widget
// can update its own view state at any time. Saved view state is all stored in single files

func saveAnalysisLayoutState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	var req analysisLayoutState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveSpectrumState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req spectrumWidgetState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveContextImageState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req contextImageState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveQuantificationState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	var req quantificationState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveHistogramState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req histogramState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveSelectionState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	req := defaultSelectionState()
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveChordState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req chordState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveBinaryState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req binaryState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveTernaryState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req ternaryState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveTableState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req tableState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveROIQuantTableState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req roiQuantTableState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveVariogramState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req variogramState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveRGBUPlotState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req rgbuPlotWidgetState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate?

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveSingleAxisRGBUState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req singleAxisRGBUWidgetState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveRGBUImagesState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req rgbuImagesWidgetState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate?

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveParallelogramState(params handlers.ApiHandlerParams, body []byte, whichInstance string) (interface{}, error) {
	// Read in body
	var req parallelogramWidgetState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate?

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveAnnotationState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	var req annotationDisplayState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func saveROIState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	var req roiDisplayState
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// TODO: Validate? Maybe make sure there are no duplicate elements, log if there are

	// Replace existing
	return nil, writeViewStateFile(params, req)
}

func writeViewStateFile(params handlers.ApiHandlerParams, reqPtr interface{}) error {
	datasetID := params.PathParams[datasetIdentifier]
	fileName := params.PathParams[idIdentifier]
	writePath := filepaths.GetViewStatePath(params.UserInfo.UserID, datasetID, fileName)

	return params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, writePath, reqPtr)
}

func saveAllState(params handlers.ApiHandlerParams, body []byte) (interface{}, error) {
	// Read in body
	var state wholeViewState

	err := json.Unmarshal(body, &state)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Delete all files that are there, because we don't want other older ones hanging around
	// This is kind of our only chance at cleanup in this space!
	clearLastViewStateFiles(params)

	// Break the incoming structure into the individual files it would be composed of if each was saved
	// individually. Remember, this is stored in individual files to allow fast per-widget saves
	// instead of saving the entire view state each time the user does something minor in a widget
	datasetID := params.PathParams[datasetIdentifier]

	// Single saves
	params.PathParams[idIdentifier] = "annotations"
	err = writeViewStateFile(params, state.Annotations)
	if err != nil {
		return nil, err
	}

	params.PathParams[idIdentifier] = "roi"
	err = writeViewStateFile(params, state.ROIs)
	if err != nil {
		return nil, err
	}

	params.PathParams[idIdentifier] = "quantification"
	err = writeViewStateFile(params, state.Quantification)
	if err != nil {
		return nil, err
	}

	params.PathParams[idIdentifier] = "selection"
	err = writeViewStateFile(params, state.Selection)
	if err != nil {
		return nil, err
	}

	// Now save the layout file.
	// NOTE: Widgets we save will be checked against this layout to ensure they exist!
	params.PathParams[idIdentifier] = "analysisLayout"
	err = writeViewStateFile(params, state.AnalysisLayout)
	if err != nil {
		return nil, err
	}

	// We may receive some widget state info for widgets that aren't actually showing (as per analysisLayout field)
	// so this filters those out
	filterUnusedWidgetStates(&state)

	widgetStructs := make(map[string]interface{})
	for k, v := range state.ContextImages {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "contextImage", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.Histograms {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "histogram", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.ChordDiagrams {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "chord", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.BinaryPlots {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "binary", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.TernaryPlots {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "ternary", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.Tables {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "table", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.ROIQuantTables {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "roiQuantTable", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.Variograms {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "variogram", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.RGBUPlots {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "rgbuPlot", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.SingleAxisRGBU {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "singleAxisRGBU", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.RGBUImageViews {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "rgbuImages", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.Parallelograms {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "parallelogram", widgetStructs)

	widgetStructs = make(map[string]interface{})
	for k, v := range state.Spectrums {
		widgetStructs[k] = v
	}
	saveWidgetStateMap(params, datasetID, "spectrum", widgetStructs)

	return nil, nil
}

func saveWidgetStateMap(params handlers.ApiHandlerParams, datasetID string, widgetName string, widgetStructs map[string]interface{}) error {
	// NOTE: Sorting is only to ensure order for unit test to pass...
	keys := []string{}
	for k := range widgetStructs {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		widgetIdentifier := k
		widgetState := widgetStructs[k]
		widgetNameAndID := widgetName + "-" + widgetIdentifier
		writePath := filepaths.GetViewStatePath(params.UserInfo.UserID, datasetID, widgetNameAndID)
		err := params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, writePath, widgetState)
		if err != nil {
			return err
		}
	}

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
