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

package output

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/pixlise/core/api/filepaths"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/data-converter/converterModels"
	protos "github.com/pixlise/core/generated-protos"
)

func makeSummaryFileContent(exp *protos.Experiment, datasetID string, group string, meta converterModels.FileMetaData, fileSize int, creationUnixTimeSec int64) datasetModel.SummaryFileData {
	contextImgCount := len(exp.AlignedContextImages) + len(exp.UnalignedContextImages) + len(exp.MatchedAlignedContextImages)
	tiffContextImgCount := 0

	// Count the number of TIFF context images so that we can quickly determine if the dataset is RGBU
	for _, img := range exp.AlignedContextImages {
		if strings.HasSuffix(img.Image, ".tif") {
			tiffContextImgCount++
		}
	}
	for _, img := range exp.UnalignedContextImages {
		if strings.HasSuffix(img, ".tif") {
			tiffContextImgCount++
		}
	}
	for _, img := range exp.MatchedAlignedContextImages {
		if strings.HasSuffix(img.Image, ".tif") {
			tiffContextImgCount++
		}
	}
	// Even if we didn't find any context images set for a given PMC, we show it as 1 because the context image
	// may have been specified as just a default file name (this is to support old test datasets that didn't have
	// the concept of PMCs, just a bunch of spectrum files and a jpeg img)
	if contextImgCount <= 0 {
		contextImgCount = 1
	}

	s := datasetModel.SummaryFileData{
		DatasetID:           datasetID,
		Group:               group,
		ContextImage:        exp.MainContextImage,
		TargetID:            exp.TargetId,
		SiteID:              meta.SiteID,
		DriveID:             meta.DriveID,
		Target:              meta.Target,
		Site:                meta.Site,
		Title:               meta.Title,
		SOL:                 meta.SOL,
		RTT:                 meta.RTT,
		SCLK:                meta.SCLK,
		LocationCount:       len(exp.Locations),
		DataFileSize:        fileSize,
		ContextImages:       contextImgCount,
		TIFFContextImages:   tiffContextImgCount,
		NormalSpectra:       int(exp.NormalSpectra),
		DwellSpectra:        int(exp.DwellSpectra),
		BulkSpectra:         int(exp.BulkSpectra),
		MaxSpectra:          int(exp.MaxSpectra),
		PseudoIntensities:   int(exp.PseudoIntensities),
		DetectorConfig:      exp.DetectorConfig,
		CreationUnixTimeSec: creationUnixTimeSec,
	}
	return s
}

func SummaryDiff(summaryFile datasetModel.SummaryFileData, bucket string, fs fileaccess.FileAccess) (datasetModel.SummaryFileData, error) {
	// TODO refactor summary file content out of output to make the diff saner

	// Create a new Summary struct for collecting the diff
	summaryDiffData := datasetModel.SummaryFileData{}
	ptrDiff := &summaryDiffData
	valuesDiff := reflect.ValueOf(ptrDiff)

	// Query existing Summary Data and reflect values
	oldSummary, err := lookUpPreviousSummary(summaryFile.RTT, bucket, fs)
	ptr := &oldSummary
	if err != nil {
		return summaryDiffData, err
	}
	valuesOld := reflect.ValueOf(ptr)

	// Prep fields and values of the new Summary struct for iteration
	fields := reflect.TypeOf(summaryFile)
	values := reflect.ValueOf(summaryFile)
	num := fields.NumField()

	for i := 0; i < num; i++ {
		// Grab the value for new, old and diff structs based on the current field
		field := fields.Field(i)
		value := values.Field(i)
		valueOld := reflect.Indirect(valuesOld).FieldByName(field.Name)
		valueDiff := reflect.Indirect(valuesDiff).FieldByName(field.Name)
		//fmt.Print("Type:", field.Type, ",", field.Name, "=", value, "\n")

		// If field has changed, indicate the new value in diff, otherwise leave diff field nil
		switch value.Kind() {
		case reflect.String:
			if value.String() != valueOld.String() {
				valueDiff.SetString(value.String())
			}
			//fmt.Print(value, "\n")
		case reflect.Int, reflect.Int32, reflect.Int64:
			if value.Int() != valueOld.Int() {
				valueDiff.SetInt(value.Int())
			}
			//fmt.Print(strconv.FormatInt(value.Int(), 10), "\n")
		case reflect.Slice:
			//
			_, ok := value.Interface().([]string)
			if !ok {
				fmt.Println("Couldn't parse slice")
			}
			//fmt.Printf("%v", len(slice))
		case reflect.Struct:
			reflect.DeepEqual(value, valueOld)
		default:
			return summaryDiffData, fmt.Errorf("unable to compare field of type: %s", value.Kind())
		}
	}

	return summaryDiffData, nil
}

func lookUpPreviousSummary(rtt int32, bucket string, fs fileaccess.FileAccess) (datasetModel.SummaryFileData, error) {
	summaryData := datasetModel.SummaryFileData{}
	// Listing files in artifact bucket/<dataset-id>/
	datasetID := fmt.Sprintf("%v", rtt)
	files, err := fs.ListObjects(bucket, datasetID)
	if err != nil {
		return summaryData, err
	}
	if len(files) > 0 {
		err = fs.ReadJSON(bucket, path.Join(datasetID, filepaths.DatasetSummaryFileName), &summaryData, false)
		if err != nil {
			return summaryData, err
		}
	}
	return summaryData, nil
}
