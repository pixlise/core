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

package endpoints

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/utils"
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
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+savedURIPath, datasetIdentifier, idIdentifier)+"/rename", apiRouter.MakeMethodPermission("POST", permission.PermWritePIXLISESettings), savedViewStateRenamePost)
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
		err := clearViewStateFiles(params)
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

func clearViewStateFiles(params handlers.ApiHandlerParams) error {
	datasetID := params.PathParams[datasetIdentifier]

	// List all files in the path
	listing, err := params.Svcs.FS.ListObjects(
		params.Svcs.Config.UsersBucket,
		filepaths.GetViewStatePath(params.UserInfo.UserID, datasetID, "")+"/",
	)
	if err != nil {
		return err
	}

	// Delete them all
	fails := []string{}
	for _, item := range listing {
		err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, item)
		if err != nil {
			fails = append(fails, item)
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

	// Get the widget type/instance
	dataType, whichInstance := splitWidgetFileName(params.PathParams[idIdentifier])

	// For every widget, we have a separate save method
	// First try saving the ones that are singular

	if len(whichInstance) <= 0 {
		switch dataType {
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
