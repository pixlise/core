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

package output

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/pixlise/core/v2/api/filepaths"
	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	protos "github.com/pixlise/core/v2/generated-protos"
)

func makeSummaryFileContent(exp *protos.Experiment, datasetID string, group string, meta dataConvertModels.FileMetaData, fileSize int, creationUnixTimeSec int64) datasetModel.SummaryFileData {
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
	oldSummary, err := lookUpPreviousSummary(summaryFile.GetRTT(), bucket, fs)
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

func lookUpPreviousSummary(datasetID string, bucket string, fs fileaccess.FileAccess) (datasetModel.SummaryFileData, error) {
	summaryData := datasetModel.SummaryFileData{}

	// Listing files in artifact bucket/<dataset-id>/
	files, err := fs.ListObjects(bucket, datasetID)
	if err != nil {
		return summaryData, err
	}
	if len(files) > 0 {
		err = fs.ReadJSON(bucket, path.Join(datasetID, filepaths.DatasetSummaryFileName), &summaryData, false)
		summaryData.RTT = summaryData.GetRTT() // For backwards compatibility we can read it as an int, but here we convert to string
		if err != nil {
			return summaryData, err
		}
	}
	return summaryData, nil
}
