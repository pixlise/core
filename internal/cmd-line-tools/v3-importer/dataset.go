package main

import (
	"context"
	"fmt"
	"path"
	"sync"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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

func SrcGetDatasetFilePath(datasetID string, fileName string) string {
	return path.Join("Datasets", datasetID, fileName)
}

func migrateDatasets(
	configBucket string,
	srcBucket string,
	destDataBucket string,
	fs fileaccess.FileAccess,
	dest *mongo.Database,
	limitToDatasetIds []string,
	userGroups map[string]string) error {
	// Drop images collection
	coll := dest.Collection(dbCollections.ImagesName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	// And beam locations
	coll = dest.Collection(dbCollections.ImageBeamLocationsName)
	err = coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	// And default images
	coll = dest.Collection(dbCollections.ScanDefaultImagesName)
	err = coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	// Also drop scan collection
	coll = dest.Collection(dbCollections.ScansName)
	err = coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	// First, get the dataset summaries
	summaries := SrcDatasetConfig{}
	err = fs.ReadJSON(configBucket, "PixliseConfig/datasets.json", &summaries, false)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	insertCount := 0

	for _, dataset := range summaries.Datasets {
		if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(dataset.DatasetID, limitToDatasetIds) {
			fmt.Printf(" SKIPPING scan: %v...\n", dataset.DatasetID)
			continue
		}

		wg.Add(1)
		go func(dataset SrcSummaryFileData) {
			defer wg.Done()
			fmt.Printf("Importing scan: %v...\n", dataset.DatasetID)

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
				TimestampUnixSec: uint32(dataset.CreationUnixTimeSec),
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

			// Decide which group to link this scan to
			memberGroup := userGroups["PIXL-FM"]
			if instrument == protos.ScanInstrument_PIXL_EM {
				memberGroup = userGroups["PIXL-EM"]
			} else if instrument != protos.ScanInstrument_PIXL_FM {
				memberGroup = userGroups["JPL Breadboard"]
			}

			_, err := coll.InsertOne(context.TODO(), &destItem)
			if err != nil {
				fatalError(err)
			}

			// Each scan needs an ownership item to define who can view/edit it
			// Prefix the ID with "scan_" because the dataset IDs are likely not that long, and we also want them
			// to differ from our random ones
			err = saveOwnershipItem( /*"scan_"+*/ dataset.DatasetID, protos.ObjectType_OT_SCAN, "", memberGroup, "", uint32(dataset.CreationUnixTimeSec), dest)
			if err != nil {
				fatalError(err)
			}

			err = importImagesForDataset(dataset.DatasetID, instrument, srcBucket, destDataBucket, fs, dest)
			if err != nil {
				fatalError(err)
			}

			// Copy dataset bin file
			s3SourcePath := SrcGetDatasetFilePath(dataset.DatasetID, filepaths.DatasetFileName)
			s3DestPath := filepaths.GetScanFilePath(dataset.DatasetID, filepaths.DatasetFileName)
			err = fs.CopyObject(srcBucket, s3SourcePath, destDataBucket, s3DestPath)
			if err != nil {
				fatalError(err)
			}

			// Copy diffraction db bin file
			s3SourcePath = SrcGetDatasetFilePath(dataset.DatasetID, filepaths.DiffractionDBFileName)
			s3DestPath = filepaths.GetScanFilePath(dataset.DatasetID, filepaths.DiffractionDBFileName)
			err = fs.CopyObject(srcBucket, s3SourcePath, destDataBucket, s3DestPath)
			if err != nil {
				fatalError(err)
			}

			// Set the default image
			err = setDefaultImage(dataset.DatasetID, dataset.ContextImage, dest)
			if err != nil {
				fatalError(err)
			}

			insertCount++
		}(dataset)
	}

	// Wait for all
	wg.Wait()

	fmt.Printf("Scans inserted: %v\n", insertCount)
	return err
}

func setDefaultImage(scanId string, image string, db *mongo.Database) error {
	if len(image) <= 0 {
		return nil // nothing to store
	}

	coll := db.Collection(dbCollections.ScanDefaultImagesName)

	// Check if we need to prefix the name
	imageSaveName := getImageSaveName(scanId, image)

	_, err := coll.InsertOne(context.TODO(), &protos.ScanImageDefaultDB{ScanId: scanId, DefaultImageFileName: imageSaveName})
	return err
}
