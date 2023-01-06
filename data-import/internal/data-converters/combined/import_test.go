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

package combined

import (
	"fmt"
	"sort"

	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	protos "github.com/pixlise/core/v2/generated-protos"
)

func Example_makeDatasetPMCOffsets() {
	datasets := map[string]*dataConvertModels.OutputData{
		"345": {
			DatasetID: "345",
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				6000: {
					SourceRTT: "345",
				},
				200: {
					SourceRTT: "345",
				},
			},
		},
		"678": {
			DatasetID: "678",
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				200: {
					SourceRTT: "678",
				},
				550: {
					SourceRTT: "678",
				},
			},
		},
		"123": {
			DatasetID: "123",
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				11000: {
					SourceRTT: "123",
				},
				13000: {
					SourceRTT: "123",
				},
			},
		},
		"567": {
			DatasetID: "567",
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				300: {
					SourceRTT: "567",
				},
				52500: {
					SourceRTT: "567",
				},
			},
		},
	}

	ids, offsets := makeDatasetPMCOffsets(datasets)
	fmt.Printf("ids: %v\noffsets: %v\n", ids, offsets)

	// Output:
	// ids: [345 678 567 123]
	// offsets: [0 10000 20000 70000]
}

func Example_combineDatasets() {
	out, err := combineDatasets(map[string]*dataConvertModels.OutputData{
		"123": {
			DatasetID:           "123",
			Group:               "PIXL-FM",
			DefaultContextImage: "45.png",
			Meta: dataConvertModels.FileMetaData{
				SCLK:  1234,
				SOL:   "500",
				RTT:   "123",
				Title: "Dataset 123",
			},
			HousekeepingHeaders: []string{"One", "Two", "Three"},
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				32: {
					SourceRTT:              "123",
					HousekeepingHeaderIdxs: []int32{0, 1, 2},
					Housekeeping: []dataConvertModels.MetaValue{
						{
							SValue:   "value1",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value2",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value3",
							DataType: protos.Experiment_MT_STRING,
						},
					},
					Beam: &dataConvertModels.BeamLocation{
						X:        1,
						Y:        2,
						Z:        3,
						GeomCorr: 0.1,
						IJ: map[int32]dataConvertModels.BeamLocationProj{
							45: {I: 100, J: 200},
						},
					},
				},
				45: {
					SourceRTT:              "123",
					ContextImageSrc:        "45.tif",
					ContextImageDst:        "45.png",
					HousekeepingHeaderIdxs: []int32{0, 1, 2},
					Housekeeping: []dataConvertModels.MetaValue{
						{
							SValue:   "value4",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value5",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value6",
							DataType: protos.Experiment_MT_STRING,
						},
					},
					Beam: &dataConvertModels.BeamLocation{
						X:        2,
						Y:        3,
						Z:        4,
						GeomCorr: 0.2,
						IJ: map[int32]dataConvertModels.BeamLocationProj{
							45: {I: 101, J: 202},
						},
					},
				},
			},
		},
		"456": {
			DatasetID:           "456",
			Group:               "PIXL-FM",
			DefaultContextImage: "88.png",
			Meta: dataConvertModels.FileMetaData{
				SCLK:  4567,
				SOL:   "600",
				RTT:   "456",
				Title: "Dataset 456",
			},
			HousekeepingHeaders: []string{"Two", "One", "Four"},
			PerPMCData: map[int32]*dataConvertModels.PMCData{
				88: {
					SourceRTT:              "456",
					ContextImageSrc:        "88.tif",
					ContextImageDst:        "88.png",
					HousekeepingHeaderIdxs: []int32{0, 1, 2},
					Housekeeping: []dataConvertModels.MetaValue{
						{
							SValue:   "value7",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value8",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value9",
							DataType: protos.Experiment_MT_STRING,
						},
					},
					Beam: &dataConvertModels.BeamLocation{
						X:        3,
						Y:        4,
						Z:        5,
						GeomCorr: 0.3,
						IJ: map[int32]dataConvertModels.BeamLocationProj{
							88: {I: 103, J: 204},
						},
					},
				},
				95: {
					SourceRTT:              "456",
					HousekeepingHeaderIdxs: []int32{0, 1, 2},
					Housekeeping: []dataConvertModels.MetaValue{
						{
							SValue:   "value10",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value11",
							DataType: protos.Experiment_MT_STRING,
						},
						{
							SValue:   "value12",
							DataType: protos.Experiment_MT_STRING,
						},
					},
					Beam: &dataConvertModels.BeamLocation{
						X:        4,
						Y:        5,
						Z:        6,
						GeomCorr: 0.4,
						IJ: map[int32]dataConvertModels.BeamLocationProj{
							88: {I: 105, J: 206},
						},
					},
				},
			},
		},
	}, &logger.StdOutLoggerForTest{})

	// Print out things we care about
	fmt.Printf("error %v\nmeta %+v\nsources %+v\nhousekeeping %v\n", err, out.Meta, out.Sources, out.HousekeepingHeaders)

	// Print them in order so it won't fail randomly
	pmcs := []int{}
	for pmc := range out.PerPMCData {
		pmcs = append(pmcs, int(pmc))
	}
	sort.Ints(pmcs)

	for _, pmc := range pmcs {
		data := out.PerPMCData[int32(pmc)]
		fmt.Printf("%v: src=%v\n context img: %v->%v\n housekeeping idxs: %+v\n housekeeping values: %+v\n", pmc, data.SourceRTT, data.ContextImageSrc, data.ContextImageDst, data.HousekeepingHeaderIdxs, data.Housekeeping)
		if data.Beam != nil {
			fmt.Printf(" beam: %+v\n", data.Beam)
		}
	}

	// Output:
	// error <nil>
	// meta {RTT:123+456 SCLK:1234 SOL:500 SiteID:0 Site: DriveID:0 TargetID: Target: Title:Combined Dataset 123+Dataset 456 Instrument:}
	// sources [{RTT:123 SCLK:1234 SOL:500 SiteID:0 Site: DriveID:0 TargetID: Target: Title:Dataset 123 Instrument:} {RTT:456 SCLK:4567 SOL:600 SiteID:0 Site: DriveID:0 TargetID: Target: Title:Dataset 456 Instrument:}]
	// housekeeping [One Two Three Four]
	// 32: src=123
	//  context img: ->
	//  housekeeping idxs: [0 1 2]
	//  housekeeping values: [{SValue:value1 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value2 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value3 IValue:0 FValue:0 DataType:MT_STRING}]
	//  beam: &{X:1 Y:2 Z:3 GeomCorr:0.1 IJ:map[45:{I:100 J:200}]}
	// 45: src=123
	//  context img: 45.tif->45.png
	//  housekeeping idxs: [0 1 2]
	//  housekeeping values: [{SValue:value4 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value5 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value6 IValue:0 FValue:0 DataType:MT_STRING}]
	//  beam: &{X:2 Y:3 Z:4 GeomCorr:0.2 IJ:map[45:{I:101 J:202}]}
	// 10088: src=456
	//  context img: 88.tif->88.png
	//  housekeeping idxs: [1 0 3]
	//  housekeeping values: [{SValue:value7 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value8 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value9 IValue:0 FValue:0 DataType:MT_STRING}]
	//  beam: &{X:3 Y:4 Z:5 GeomCorr:0.3 IJ:map[88:{I:103 J:204}]}
	// 10095: src=456
	//  context img: ->
	//  housekeeping idxs: [1 0 3]
	//  housekeeping values: [{SValue:value10 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value11 IValue:0 FValue:0 DataType:MT_STRING} {SValue:value12 IValue:0 FValue:0 DataType:MT_STRING}]
	//  beam: &{X:4 Y:5 Z:6 GeomCorr:0.4 IJ:map[88:{I:105 J:206}]}
}
