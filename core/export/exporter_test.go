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

package export

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"math"
	"path"
	"testing"

	datasetModel "github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/timestamper"
	"github.com/pixlise/core/v2/core/utils"
)

func Test_makeMarkupImage(t *testing.T) {
	img, err := utils.ReadImageFile("./test-data/MCC-67.png")
	if err != nil {
		t.Errorf("%v", err)
	}

	ds, err := datasetModel.ReadDatasetFile("./test-data/dataset.bin")
	if err != nil {
		t.Errorf("%v", err)
	}

	outImg, err := makeMarkupImage(img, -1, ds, nil)
	if err != nil {
		t.Errorf("%v", err)
	}

	// Save it so we can compare
	err = utils.WritePNGImageFile("/tmp/markup", outImg)
	if err != nil {
		t.Errorf("%v", err)
	}

	err = utils.ImagesEqual("./test-data/expected-markup-MCC-67.png", "/tmp/markup.png")
	if err != nil {
		t.Errorf("Output markup image does not match expected: %v", err)
	}
}

func Test_makeBeamLocationCSV(t *testing.T) {
	ds, err := datasetModel.ReadDatasetFile("./test-data/dataset.bin")
	if err != nil {
		t.Errorf("%v", err)
	}

	beams := datasetModel.MakePMCBeamLookup(ds)

	err = writeBeamCSV("/tmp/", "test-name", beams, ds)
	if err != nil {
		t.Errorf("Failed to write beam CSV: %v", err)
	}

	err = utils.FilesEqual("/tmp/test-name-beam-locations.csv", "./test-data/expected-beams.csv")
	if err != nil {
		t.Errorf("Incorrect beam CSV: %v", err)
	}
}

func Test_makeROICSV(t *testing.T) {
	ds, err := datasetModel.ReadDatasetFile("./test-data/dataset.bin")
	if err != nil {
		t.Errorf("%v", err)
	}

	obj := pixlUser.APIObjectItem{
		Shared: false,
		Creator: pixlUser.UserInfo{
			Name:        "name",
			UserID:      "user_id",
			Email:       "email",
			Permissions: map[string]bool{},
		},
	}

	userROIs := roiModel.ROILookup{}
	userROIs["roi123"] = roiModel.ROISavedItem{
		ROIItem: &roiModel.ROIItem{
			Name:            "something",
			LocationIndexes: []int32{0, 1, 2, 4},
		},
		APIObjectItem: &obj,
	}
	userROIs["roi456"] = roiModel.ROISavedItem{
		ROIItem: &roiModel.ROIItem{
			Name:            "second",
			LocationIndexes: []int32{3, 4, 5},
		},
		APIObjectItem: &obj,
	}

	sharedROIs := roiModel.ROILookup{}
	sharedROIs["shared111"] = roiModel.ROISavedItem{
		ROIItem: &roiModel.ROIItem{
			Name:            "le shared",
			LocationIndexes: []int32{5, 6, 8, 100},
		},
		APIObjectItem: &obj,
	}

	rois := roiModel.GetROIsWithPMCs(userROIs, sharedROIs, ds)

	err = writeROICSV("/tmp/", rois)
	if err != nil {
		t.Errorf("Failed to write ROI CSV: %v", err)
	}

	err = utils.FilesEqual("/tmp/something-roi-pmcs.csv", "./test-data/expected-something-rois.csv")
	if err != nil {
		t.Errorf("Incorrect ROI CSV: %v", err)
	}

	err = utils.FilesEqual("/tmp/second-roi-pmcs.csv", "./test-data/expected-second-rois.csv")
	if err != nil {
		t.Errorf("Incorrect ROI CSV: %v", err)
	}

	err = utils.FilesEqual("/tmp/le_shared-roi-pmcs.csv", "./test-data/expected-le_shared-rois.csv")
	if err != nil {
		t.Errorf("Incorrect ROI CSV: %v", err)
	}
}

func Test_unquantifiedMap(t *testing.T) {
	ds, err := datasetModel.ReadDatasetFile("./test-data/dataset.bin")
	if err != nil {
		t.Errorf("%v", err)
	}

	beams := datasetModel.MakePMCBeamLookup(ds)

	q, err := quantModel.ReadQuantificationFile("./test-data/quantification.bin")
	if err != nil {
		t.Errorf("%v", err)
	}

	elemCols := []string{"Ti_%", "Cr_%", "Ni_%", "Si_%"}
	unquantWeightPct := []map[int32]float32{}
	unquantWeightPctDetector := []string{}
	for detectorIdx, locSet := range q.LocationSet {
		unquant, err := makeUnquantifiedMapValues(beams, q, detectorIdx, elemCols)
		if err != nil {
			t.Errorf("%v", err)
		}
		unquantWeightPct = append(unquantWeightPct, unquant)
		unquantWeightPctDetector = append(unquantWeightPctDetector, locSet.Detector)
	}

	if len(unquantWeightPctDetector) != 2 || unquantWeightPctDetector[0] != "A" || unquantWeightPctDetector[1] != "B" {
		t.Errorf("Invalid detector list generated")
	}

	//fmt.Printf("%v", unquantWeightPct[0])

	if len(unquantWeightPct) != 2 {
		t.Errorf("Not enough data per detector")
	}

	// Verify a few values in the quant file, so we know we're reading the right things...
	if q.LocationSet[0].Detector != "A" {
		t.Errorf("Expected detector A in source file, got %v", q.LocationSet[0].Detector)
	}

	if q.LocationSet[1].Detector != "B" {
		t.Errorf("Expected detector B in source file, got %v", q.LocationSet[1].Detector)
	}

	if q.Labels[0] != "Si_%" || q.Labels[2] != "Ti_%" || q.Labels[4] != "Cr_%" || q.Labels[6] != "Ni_%" {
		t.Errorf("Expected element order incorrect in source quant file")
	}

	for detIdx := int32(0); detIdx < 2; detIdx++ {
		detector := "A"
		if detIdx > 0 {
			detector = "B"
		}

		const minPMC = int32(67)
		const maxPMC = int32(376)

		for c := range q.LocationSet[detIdx].Location {
			loc := q.LocationSet[detIdx].Location[c]
			pmc := loc.Pmc

			expVal := 100 -
				loc.Values[0].Fvalue -
				loc.Values[2].Fvalue -
				loc.Values[4].Fvalue -
				loc.Values[6].Fvalue
			if math.Abs(float64(unquantWeightPct[detIdx][pmc]-expVal)) > 0.0001 {
				t.Errorf("Unquantified pct value for PMC=%v, Det=%v expected: %v, got %v", pmc, detector, expVal, unquantWeightPct[detIdx][pmc])
			}
			// Negative values are a valid possibility if quant file had values that add up to > 100!
			/*
				if unquantWeightPct[detIdx][pmc] < 0 {
					t.Errorf("Unquantified pct value is %v for PMC=%v, Det=%v. Si=%v, Ti=%v, Cr=%v, Ni=%v", unquantWeightPct[detIdx][pmc], pmc, detector, loc.Values[0].Fvalue, loc.Values[2].Fvalue, loc.Values[4].Fvalue, loc.Values[6].Fvalue)
				}
			*/
		}
	}

	err = writeUnquantifiedWeightPctCSV("/tmp/", "testname", unquantWeightPctDetector, unquantWeightPct)
	if err != nil {
		t.Errorf("Failed to write unquantified CSV: %v", err)
	}

	err = utils.FilesEqual("/tmp/testname-unquantified-weight-pct.csv", "./test-data/expected-unquantified.csv")
	if err != nil {
		t.Errorf("Incorrect unquantified CSV: %v", err)
	}
}

func Example_writeQuantCSVForROI() {
	quantCSVLines := []string{"header", "PMC, Ca_%, Fe_%, livetime", "12, 5.5, 6.6, 9.8", "14, 7.7, 8.8, 9.7", "15, 2.7, 2.8, 9.6"}
	roi := roiModel.ROIMembers{
		Name:         "roi name",
		ID:           "roi123",
		SharedByName: "",
		LocationIdxs: []int32{4, 5, 6},
		PMCs:         []int32{11, 12, 13, 14},
	}
	outDir, err := ioutil.TempDir("", "csv-test")
	if err != nil {
		fmt.Printf("Failed to make temp dir: %v\n", err)
	}
	csvName, err := writeQuantCSVForROI(quantCSVLines, roi, outDir, "prefix")
	fmt.Printf("%v|%v\n", err, csvName)

	// Write the file to stdout so we can test its contents
	data, err := ioutil.ReadFile(path.Join(outDir, csvName))
	if err != nil {
		fmt.Printf("Failed to read csv: %v\n", err)
	}
	fmt.Println(string(data))

	// Output:
	// <nil>|prefix-map ROI roi name.csv
	// header
	// PMC, Ca_%, Fe_%, livetime
	// 12, 5.5, 6.6, 9.8
	// 14, 7.7, 8.8, 9.7
}

type stringMemWriter struct {
	Content *string
}

func (w stringMemWriter) WriteString(s string) (n int, err error) {
	(*w.Content) += s
	return len(s), nil
}

func Example_writeSpectraCSV_OK() {
	spectra := []spectrumData{
		{
			PMC: 12,
			x:   1,
			y:   2,
			z:   3,
			metaA: datasetModel.SpectrumMetaValues{
				SCLK:     12345678,
				RealTime: 8.3,
				LiveTime: 8.0,
				XPerChan: 10.1,
				Offset:   2.7,
				Detector: "A",
				ReadType: "Normal",
			},
			metaB: datasetModel.SpectrumMetaValues{
				SCLK:     123456789,
				RealTime: 8.2,
				LiveTime: 8.5,
				XPerChan: 10.6,
				Offset:   3.7,
				Detector: "B",
				ReadType: "Normal",
			},
			countsA: []int32{30, 500, 10},
			countsB: []int32{40, 300, 2},
		},
	}

	var content = ""
	csv := stringMemWriter{Content: &content}
	err := writeSpectraCSV("the/path.csv", spectra, csv)
	fmt.Println(*csv.Content)
	fmt.Printf("%v\n", err)

	// Output:
	// SCLK_A,SCLK_B,PMC,real_time_A,real_time_B,live_time_A,live_time_B,XPERCHAN_A,XPERCHAN_B,OFFSET_A,OFFSET_B
	// 12345678,123456789,12,8.3,8.2,8,8.5,10.1,10.6,2.7,3.7
	// PMC,x,y,z
	// 12,1,2,3
	// A_1,A_2,A_3
	// 30,500,10
	// B_1,B_2,B_3
	// 40,300,2
	//
	// <nil>
}

func Example_writeSpectraCSV_Blank() {
	var content = ""
	csv := stringMemWriter{Content: &content}
	err := writeSpectraCSV("the/path.csv", []spectrumData{}, csv)
	fmt.Println(*csv.Content)
	fmt.Printf("%v\n", err)

	// Output:
	// No spectra for writeSpectraCSV when writing the/path.csv
}

func Example_writeSpectraMSA_AB() {
	spectrum := spectrumData{
		PMC: 12,
		x:   1,
		y:   2,
		z:   3,
		metaA: datasetModel.SpectrumMetaValues{
			SCLK:     12345678,
			RealTime: 8.3,
			LiveTime: 8.0,
			XPerChan: 10.1,
			Offset:   2.7,
			Detector: "A",
			ReadType: "Normal",
		},
		metaB: datasetModel.SpectrumMetaValues{
			SCLK:     123456789,
			RealTime: 8.2,
			LiveTime: 8.5,
			XPerChan: 10.6,
			Offset:   3.7,
			Detector: "B",
			ReadType: "Normal",
		},
		countsA: []int32{30, 500, 10},
		countsB: []int32{40, 300, 2},
	}

	var content = ""
	msa := stringMemWriter{Content: &content}
	mockTime := &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1734567890},
	}
	err := writeSpectraMSA("the/path.msa", mockTime, spectrum, msa)
	fmt.Println(*msa.Content)
	fmt.Printf("%v\n", err)

	// Output:
	// #FORMAT      : EMSA/MAS spectral data file
	// #VERSION     : TC202v2.0 PIXL
	// #TITLE       : Control Program v7
	// #OWNER       : JPL BREADBOARD vx
	// #DATE        : 12-19-2024
	// #TIME        : 00:24:50
	// #NPOINTS     : 3
	// #NCOLUMNS    : 2
	// #XUNITS      :  eV
	// #YUNITS      :  COUNTS
	// #DATATYPE    :  YY
	// #XPERCHAN    :  10.1, 10.6    eV per channel
	// #OFFSET      :  2.7, 3.7    eV of first channel
	// #SIGNALTYPE  :  XRF
	// #COMMENT     :  Exported bulk sum MSA from PIXLISE
	// #XPOSITION   :    0.000
	// #YPOSITION   :    0.000
	// #ZPOSITION   :    0.000
	// #LIVETIME    :  8, 8.5
	// #REALTIME    :  8.3, 8.2
	// #SPECTRUM    :
	// 30, 40
	// 500, 300
	// 10, 2
	//
	// <nil>
}

func Example_writeSpectraMSA_A() {
	spectrum := spectrumData{
		PMC: 12,
		x:   1,
		y:   2,
		z:   3,
		metaA: datasetModel.SpectrumMetaValues{
			SCLK:     12345678,
			RealTime: 8.3,
			LiveTime: 8.0,
			XPerChan: 10.1,
			Offset:   2.7,
			Detector: "A",
			ReadType: "Normal",
		},
		countsA: []int32{30, 500, 10},
	}

	var content = ""
	msa := stringMemWriter{Content: &content}
	mockTime := &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1734567890},
	}
	err := writeSpectraMSA("the/path.msa", mockTime, spectrum, msa)
	fmt.Println(*msa.Content)
	fmt.Printf("%v\n", err)

	// Output:
	// #FORMAT      : EMSA/MAS spectral data file
	// #VERSION     : TC202v2.0 PIXL
	// #TITLE       : Control Program v7
	// #OWNER       : JPL BREADBOARD vx
	// #DATE        : 12-19-2024
	// #TIME        : 00:24:50
	// #NPOINTS     : 3
	// #NCOLUMNS    : 1
	// #XUNITS      :  eV
	// #YUNITS      :  COUNTS
	// #DATATYPE    :  Y
	// #XPERCHAN    :  10.1    eV per channel
	// #OFFSET      :  2.7    eV of first channel
	// #SIGNALTYPE  :  XRF
	// #COMMENT     :  Exported bulk sum MSA from PIXLISE
	// #XPOSITION   :    0.000
	// #YPOSITION   :    0.000
	// #ZPOSITION   :    0.000
	// #LIVETIME    :  8
	// #REALTIME    :  8.3
	// #SPECTRUM    :
	// 30
	// 500
	// 10
	//
	// <nil>
}
