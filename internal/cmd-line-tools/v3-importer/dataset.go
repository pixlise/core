package main

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

/* An example:
{
	"dataset_id": "089063943",
	"group": "PIXL-FM",
	"drive_id": 0,
	"site_id": 8,
	"target_id": "?",
	"site": "",
	"target": "",
	"title": "Dourbes",
	"sol": "0257",
	"rtt": "089063943",
	"sclk": 689790062,
	"context_image": "PCW_0257_0689790669_000RCM_N00800000890639430006075J01.png",
	"location_count": 3341,
	"data_file_size": 20767829,
	"context_images": 37,
	"tiff_context_images": 2,
	"normal_spectra": 6666,
	"dwell_spectra": 0,
	"bulk_spectra": 2,
	"max_spectra": 2,
	"pseudo_intensities": 3333,
	"detector_config": "PIXL",
	"create_unixtime_sec": 1663283056
}
*/

type SrcSummaryFileData struct {
	DatasetID           string      `json:"dataset_id"`
	Group               string      `json:"group"`
	DriveID             int32       `json:"drive_id"`
	SiteID              int32       `json:"site_id"`
	TargetID            string      `json:"target_id"`
	Site                string      `json:"site"`
	Target              string      `json:"target"`
	Title               string      `json:"title"`
	SOL                 string      `json:"sol"`
	RTT                 interface{} `json:"rtt,string"` // Unfortunately we stored it as int initially, so this has to accept files stored that way
	SCLK                int32       `json:"sclk"`
	ContextImage        string      `json:"context_image"`
	LocationCount       int         `json:"location_count"`
	DataFileSize        int         `json:"data_file_size"`
	ContextImages       int         `json:"context_images"`
	TIFFContextImages   int         `json:"tiff_context_images"`
	NormalSpectra       int         `json:"normal_spectra"`
	DwellSpectra        int         `json:"dwell_spectra"`
	BulkSpectra         int         `json:"bulk_spectra"`
	MaxSpectra          int         `json:"max_spectra"`
	PseudoIntensities   int         `json:"pseudo_intensities"`
	DetectorConfig      string      `json:"detector_config"`
	CreationUnixTimeSec int64       `json:"create_unixtime_sec"`
}

func (s SrcSummaryFileData) GetRTT() string {
	result := ""
	switch s.RTT.(type) {
	case float64:
		f, ok := s.RTT.(float64)
		if ok {
			result = fmt.Sprintf("%d", int(f))
		}
	case int:
		i, ok := s.RTT.(int)
		if ok {
			result = fmt.Sprintf("%d", i)
		}
	default:
		result = fmt.Sprintf("%v", s.RTT)
	}

	padding := 9 - len(result)
	if padding > 0 {
		for i := 0; i < padding; i++ {
			result = "0" + result
		}
	}
	return result
}

type SrcDatasetConfig struct {
	Datasets []SrcSummaryFileData `json:"datasets"`
}

func migrateDatasets(configBucket string, dataBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	const collectionName = "scans"

	err := dest.Collection(collectionName).Drop(context.TODO())
	if err != nil {
		return err
	}

	// First, get the dataset summaries
	summaries := SrcDatasetConfig{}
	err = fs.ReadJSON(configBucket, "PixliseConfig/datasets.json", &summaries, false)
	if err != nil {
		return err
	}

	destItems := []interface{}{}
	for _, dataset := range summaries.Datasets {
		instrument := protos.ScanInstrument_PIXL_FM
		if dataset.Group == "PIXL_EM" {
			instrument = protos.ScanInstrument_PIXL_EM
		} else if dataset.Group == "Breadboard" {
			instrument = protos.ScanInstrument_JPL_BREADBOARD
		}

		meta := map[string]string{
			"RTT":     dataset.GetRTT(),
			"SCLK":    fmt.Sprintf("%v", dataset.SCLK),
			"Sol":     dataset.SOL,
			"DriveId": fmt.Sprintf("%v", dataset.DriveID),
			//"Drive":    dataset.Drive, <-- doesn't exist
			"TargetId": dataset.TargetID,
			"Target":   dataset.Target,
			"SiteId":   fmt.Sprintf("%v", dataset.SiteID),
			"Site":     dataset.Site,
		}

		counts := map[string]int32{
			"NormalSpectra":     int32(dataset.NormalSpectra),
			"DwellSpectra":      int32(dataset.DwellSpectra),
			"BulkSpectra":       int32(dataset.BulkSpectra),
			"MaxSpectra":        int32(dataset.MaxSpectra),
			"PseudoIntensities": int32(dataset.PseudoIntensities),
		}

		destItem := protos.ScanItem{
			Id:               dataset.DatasetID,
			Title:            dataset.Title,
			Description:      "",
			TimestampUnixSec: uint64(dataset.CreationUnixTimeSec),
			Instrument:       instrument,
			InstrumentConfig: dataset.DetectorConfig,
			DataTypes:        []*protos.ScanItem_ScanTypeCount{},
			Meta:             meta,
			ContentCounts:    counts,
		}

		if dataset.LocationCount > 0 {
			destItem.DataTypes = append(destItem.DataTypes, &protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_XRF,
				Count:    uint32(dataset.LocationCount),
			})
		}

		if dataset.TIFFContextImages > 0 {
			destItem.DataTypes = append(destItem.DataTypes, &protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_RGBU,
				Count:    uint32(dataset.TIFFContextImages),
			})
		}

		if dataset.ContextImages > 0 {
			destItem.DataTypes = append(destItem.DataTypes, &protos.ScanItem_ScanTypeCount{
				DataType: protos.ScanDataType_SD_IMAGE,
				Count:    uint32(dataset.ContextImages),
			})
		}

		destItems = append(destItems, destItem)
	}

	result, err := dest.Collection(collectionName).InsertMany(context.TODO(), destItems)
	if err != nil {
		return err
	}

	fmt.Printf("Scans inserted: %v\n", len(result.InsertedIDs))

	return err
}