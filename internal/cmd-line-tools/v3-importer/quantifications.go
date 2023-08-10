package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

const SrcQuantModeCombinedMultiQuant = "CombinedMultiQuant"
const SrcQuantModeABMultiQuant = "ABMultiQuant"

type SrcJobStartingParameters struct {
	Name              string      `json:"name"`
	DataBucket        string      `json:"dataBucket"`
	DatasetPath       string      `json:"datasetPath"`
	DatasetID         string      `json:"datasetID"`
	PiquantJobsBucket string      `json:"jobBucket"`
	DetectorConfig    string      `json:"detectorConfig"`
	Elements          []string    `json:"elements"`
	Parameters        string      `json:"parameters"`
	RunTimeSec        int32       `json:"runTimeSec"`
	CoresPerNode      int32       `json:"coresPerNode"`
	StartUnixTime     int64       `json:"startUnixTime"`
	Creator           SrcUserInfo `json:"creator"`
	RoiID             string      `json:"roiID"`
	ElementSetID      string      `json:"elementSetID"`
	PIQUANTVersion    string      `json:"piquantVersion"`
	QuantMode         string      `json:"quantMode"`
	Comments          string      `json:"comments"`
	RoiIDs            []string    `json:"roiIDs"`
	IncludeDwells     bool        `json:"includeDwells,omitempty"`
	Command           string      `json:"command,omitempty"`
}

type SrcJobStartingParametersWithPMCCount struct {
	PMCCount int32 `json:"pmcsCount"`
	*SrcJobStartingParameters
}

type SrcJobStatusValue string
type SrcJobStatus struct {
	JobID          string            `json:"jobId"`
	Status         SrcJobStatusValue `json:"status"`
	Message        string            `json:"message"`
	EndUnixTime    int64             `json:"endUnixTime"`
	OutputFilePath string            `json:"outputFilePath"`
	PiquantLogList []string          `json:"piquantLogList"`
}

type SrcJobSummaryItem struct {
	Shared   bool                                 `json:"shared"`
	Params   SrcJobStartingParametersWithPMCCount `json:"params"`
	Elements []string                             `json:"elements"`
	*SrcJobStatus
}

const (
	SrcJobStarting         SrcJobStatusValue = "starting"
	SrcJobPreparingNodes                     = "preparing_nodes"
	SrcJobNodesRunning                       = "nodes_running"
	SrcJobGatheringResults                   = "gathering_results"
	SrcJobComplete                           = "complete"
	SrcJobError                              = "error"
)

func migrateQuants(userContentBucket string, userContentFiles []string, limitToDatasetIds []string, fs fileaccess.FileAccess, dest *mongo.Database, destUserContentBucket string) error {
	coll := dest.Collection(dbCollections.QuantificationsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	destQuants := []interface{}{}
	userItems := map[string]SrcJobSummaryItem{}
	sharedItems := map[string]SrcJobSummaryItem{}

	roiSetCount := 0
	roisSetCount := 0
	elementSetSetCount := 0
	multiQuantCount := 0

	for _, p := range userContentFiles {
		if strings.Contains(p, "/Quantifications/summary-") {
			// Example: UserContent/<user-id>/<dataset-id>/Quantification/summary-<quant-id>.json
			quantId := filepath.Base(p)

			// Snip off the summary- and .json
			quantId = quantId[len("summary-") : len(quantId)-5]
			scanId := filepath.Base(filepath.Dir(filepath.Dir(p)))
			userIdFromPath := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(p))))

			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf("Skipping import of ROI from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(scanId, limitToDatasetIds) {
				fmt.Printf("Skipping quant for dataset id: %v...\n", scanId)
				continue
			}

			// Read this file
			jobSummary := SrcJobSummaryItem{}
			err := fs.ReadJSON(userContentBucket, p, &jobSummary, false)
			if err != nil {
				return err
			}

			// Update counts
			if len(jobSummary.Params.RoiID) > 0 {
				fmt.Printf("%v has RoiID set\n", p)
				roiSetCount++
			}
			if len(jobSummary.Params.RoiIDs) > 0 {
				fmt.Printf("%v has RoiIDs set\n", p)
				roisSetCount++
			}
			if len(jobSummary.Params.ElementSetID) > 0 {
				elementSetSetCount++
			}
			if jobSummary.Params.QuantMode == SrcQuantModeCombinedMultiQuant || jobSummary.Params.QuantMode == SrcQuantModeABMultiQuant {
				multiQuantCount++
			}

			// Paranoia, make sure both quant IDs match
			if jobSummary.JobID != quantId {
				return fmt.Errorf("Quant ID mismatch: path had %v, job summary file had %v. Path: %v", quantId, jobSummary.JobID, p)
			}
			if jobSummary.Params.DatasetID != scanId {
				return fmt.Errorf("Quant Scan ID mismatch: path had %v, job summary file had %v. Path: %v", scanId, jobSummary.Params.DatasetID, p)
			}

			if strings.HasPrefix(p, "UserContent/shared/") {
				// Store these till we're finished here
				sharedItems[quantId] = jobSummary
			} else {
				// Write user quant to DB and also remember them for later...
				if _, ok := userItems[quantId]; ok {
					fmt.Printf("Duplicate quantification ID: %v\n", quantId)
					continue
				}

				if jobSummary.Params.Creator.UserID != userIdFromPath {
					fmt.Printf("Unexpected quant user: %v, path had id: %v. Path was: %v\n", jobSummary.Params.Creator.UserID, userIdFromPath, p)
				}

				userItems[quantId] = jobSummary

				var jobStatus protos.JobStatus_Status

				switch jobSummary.Status {
				case SrcJobStarting:
					jobStatus = protos.JobStatus_STARTING
				case SrcJobPreparingNodes:
					jobStatus = protos.JobStatus_PREPARING_NODES
				case SrcJobNodesRunning:
					jobStatus = protos.JobStatus_NODES_RUNNING
				case SrcJobGatheringResults:
					jobStatus = protos.JobStatus_GATHERING_RESULTS
				case SrcJobComplete:
					jobStatus = protos.JobStatus_COMPLETE
				case SrcJobError:
					jobStatus = protos.JobStatus_ERROR
				}

				// Write to DB
				destQuant := &protos.QuantificationSummary{
					Id: jobSummary.JobID,
					Params: &protos.QuantStartingParametersWithPMCCount{
						PMCCount: uint32(jobSummary.Params.PMCCount),
						Params: &protos.QuantStartingParameters{
							Name:              jobSummary.Params.Name,
							DataBucket:        jobSummary.Params.DataBucket,
							DatasetPath:       jobSummary.Params.DatasetPath,
							DatasetID:         jobSummary.Params.DatasetID,
							PiquantJobsBucket: jobSummary.Params.PiquantJobsBucket,
							DetectorConfig:    jobSummary.Params.DetectorConfig,
							Elements:          jobSummary.Params.Elements,
							Parameters:        jobSummary.Params.Parameters,
							RunTimeSec:        uint32(jobSummary.Params.RunTimeSec),
							CoresPerNode:      uint32(jobSummary.Params.CoresPerNode),
							StartUnixTimeSec:  uint32(jobSummary.Params.StartUnixTime),
							RequestorUserId:   utils.FixUserId(jobSummary.Params.Creator.UserID),
							RoiID:             jobSummary.Params.RoiID,
							ElementSetID:      jobSummary.Params.ElementSetID,
							PIQUANTVersion:    jobSummary.Params.PIQUANTVersion,
							QuantMode:         jobSummary.Params.QuantMode,
							Comments:          jobSummary.Params.Comments,
							RoiIDs:            jobSummary.Params.RoiIDs,
							IncludeDwells:     jobSummary.Params.IncludeDwells,
							Command:           jobSummary.Params.Command,
						},
					},
					Elements: jobSummary.Elements,
					Status: &protos.JobStatus{
						JobID:          jobSummary.JobID,
						Status:         jobStatus,
						Message:        jobSummary.Message,
						EndUnixTimeSec: uint32(jobSummary.EndUnixTime),
						OutputFilePath: path.Join("Quantifications", scanId, utils.FixUserId(jobSummary.Params.Creator.UserID)), //jobSummary.OutputFilePath,
						PiquantLogs:    jobSummary.PiquantLogList,
					},
				}

				err = saveOwnershipItem(quantId, protos.ObjectType_OT_QUANTIFICATION, jobSummary.Params.Creator.UserID, uint32(jobSummary.EndUnixTime), dest)
				if err != nil {
					return err
				}

				// Save the relevant quantification files to their destination in S3.
				// NOTE: if they are not found, this is an error!
				saveQuantFiles(scanId, userIdFromPath, quantId, userContentBucket, destUserContentBucket, fs)

				destQuants = append(destQuants, destQuant)
			}
		}
	}

	fmt.Printf("Quantification import results: roiID field count: %v, roiIDs field count: %v, elementSetID field count: %v, multiQuants: %v\n",
		roiSetCount, roisSetCount, elementSetSetCount, multiQuantCount)

	result, err := coll.InsertMany(context.TODO(), destQuants)
	if err != nil {
		return err
	}

	fmt.Printf("Quants inserted: %v\n", len(result.InsertedIDs))

	return nil
}

func saveQuantFiles(datasetId string, userId string, quantId string, userContentBucket string, destUserContentBucket string, fs fileaccess.FileAccess) error {
	// Expecting a quant bin file
	s3Path := filepaths.GetUserQuantPath(userId, datasetId, quantId+".bin")
	bytes, err := fs.ReadObject(userContentBucket, s3Path)
	if err != nil {
		return err
	}

	// Write to new location
	s3Path = path.Join("Quantifications", datasetId, utils.FixUserId(userId), quantId+".bin")
	err = fs.WriteObject(destUserContentBucket, s3Path, bytes)
	if err != nil {
		return err
	}
	fmt.Printf("  Wrote: %v\n", s3Path)

	// Optional quant CSV file, warn if not found though
	s3Path = filepaths.GetUserQuantPath(userId, datasetId, quantId+".csv")
	bytes, err = fs.ReadObject(userContentBucket, s3Path)
	if err != nil {
		// We just warn here
		log.Printf("Failed to find quant CSV file: %v. Skipping...", s3Path)
	} else {
		// Worked, so save
		s3Path = path.Join("Quantifications", datasetId, utils.FixUserId(userId), quantId+".csv")
		err = fs.WriteObject(destUserContentBucket, s3Path, bytes)
		if err != nil {
			return err
		}
		fmt.Printf("  Wrote: %v\n", s3Path)
	}

	// Quant log files, we need a listing of these
	s3Path = filepaths.GetUserQuantPath(userId, datasetId, filepaths.MakeQuantLogDirName(quantId))
	logFiles, err := fs.ListObjects(userContentBucket, s3Path)
	if err != nil {
		// We just warn here
		log.Printf("Failed to list quant log files at: %v. Skipping...", s3Path)
	} else {
		for _, logPath := range logFiles {
			bytes, err = fs.ReadObject(userContentBucket, logPath)
			if err != nil {
				return err
			}

			s3Path = path.Join("Quantifications", datasetId, utils.FixUserId(userId), quantId+"-logs", path.Base(logPath))
			err = fs.WriteObject(destUserContentBucket, s3Path, bytes)
			if err != nil {
				return err
			}
			fmt.Printf("  Wrote: %v\n", s3Path)
		}
	}

	return nil
}

type SrcQuantCombineItem struct {
	RoiID            string `json:"roiID"`
	QuantificationID string `json:"quantificationID"`
}

type SrcQuantCombineList struct {
	RoiZStack []SrcQuantCombineItem `json:"roiZStack"`
}

const SrcMultiQuantZStackFileName = "multi-quant-z-stack.json"

func migrateMultiQuants(userContentBucket string, userContentFiles []string, limitToDatasetIds []string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.QuantificationZStacksName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	destItems := []interface{}{}

	for _, p := range userContentFiles {
		if strings.HasSuffix(p, SrcMultiQuantZStackFileName) {
			// Example: /UserContent/<user-id>/<dataset-id>/<SrcMultiQuantZStackFileName>
			datasetIdFromPath := filepath.Base(filepath.Dir(p))
			userIdFromPath := filepath.Base(filepath.Dir(filepath.Dir(p)))

			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf("Skipping import of ROI from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(datasetIdFromPath, limitToDatasetIds) {
				fmt.Printf("Skipping multi-quant for dataset id: %v...\n", datasetIdFromPath)
				continue
			}

			zstack := SrcQuantCombineList{}
			err := fs.ReadJSON(userContentBucket, p, &zstack, false)
			if err != nil {
				return err
			}

			if len(zstack.RoiZStack) > 0 {
				// Write to DB
				destZStack := []*protos.QuantCombineItem{}
				for _, item := range zstack.RoiZStack {
					destZStack = append(destZStack, &protos.QuantCombineItem{
						RoiId:            item.RoiID,
						QuantificationId: item.QuantificationID,
					})
				}

				userIdFromPath = utils.FixUserId(userIdFromPath)

				destItem := &protos.QuantCombineItemListDB{
					Id:     userIdFromPath + "_" + datasetIdFromPath,
					UserId: userIdFromPath,
					ScanId: datasetIdFromPath,
					List: &protos.QuantCombineItemList{
						RoiZStack: destZStack,
					},
				}

				destItems = append(destItems, destItem)
			}
		}
	}

	result, err := coll.InsertMany(context.TODO(), destItems)
	if err != nil {
		return err
	}

	fmt.Printf("Quant z-stacks inserted: %v\n", len(result.InsertedIDs))

	return nil
}
