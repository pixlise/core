package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
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

func migrateQuants(
	userContentBucket string,
	userContentFiles []string,
	limitToDatasetIds []string,
	fs fileaccess.FileAccess,
	dest *mongo.Database,
	destUserContentBucket string,
	userGroups map[string]string) error {
	coll := dest.Collection(dbCollections.QuantificationsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	sharedItems := map[string]SrcJobSummaryItem{}

	// First, find all the shared quants
	for _, p := range userContentFiles {
		if strings.Contains(p, "/Quantifications/summary-") && strings.HasPrefix(p, "UserContent/shared/") {
			scanId := filepath.Base(filepath.Dir(filepath.Dir(p)))

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(scanId, limitToDatasetIds) {
				fmt.Printf(" SKIPPING shared quant for dataset id: %v...\n", scanId)
				continue
			}

			// Read this file
			jobSummary := SrcJobSummaryItem{}
			err := fs.ReadJSON(userContentBucket, p, &jobSummary, false)
			if err != nil {
				return err
			}

			// Example: UserContent/<user-id>/<dataset-id>/Quantification/summary-<quant-id>.json
			quantId := filepath.Base(p)

			// Snip off the summary- and .json
			quantId = quantId[len("summary-") : len(quantId)-5]

			// Store these till we're finished here
			sharedItems[quantId] = jobSummary
		}
	}

	userItems := map[string]SrcJobSummaryItem{}

	roiSetCount := 0
	roisSetCount := 0
	elementSetSetCount := 0
	multiQuantCount := 0

	for _, p := range userContentFiles {
		if strings.Contains(p, "/Quantifications/summary-") && !strings.HasPrefix(p, "UserContent/shared/") {
			// Example: UserContent/<user-id>/<dataset-id>/Quantification/summary-<quant-id>.json
			quantId := filepath.Base(p)

			// Snip off the summary- and .json
			quantId = quantId[len("summary-") : len(quantId)-5]
			scanId := filepath.Base(filepath.Dir(filepath.Dir(p)))
			userIdFromPath := filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(p))))

			if shouldIgnoreUser(userIdFromPath) {
				fmt.Printf(" SKIPPING import of quant from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(scanId, limitToDatasetIds) {
				fmt.Printf(" SKIPPING quant for dataset id: %v...\n", scanId)
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

			// Write user quant to DB and also remember them for later...
			if _, ok := userItems[quantId]; ok {
				fmt.Printf("Duplicate quantification ID: %v\n", quantId)
				continue
			}

			if jobSummary.Params.Creator.UserID != userIdFromPath {
				fmt.Printf("Unexpected quant user: %v, path had id: %v. Path was: %v. Quant was likely copied to another user, skipping...\n" /*. Using quant user in output paths...\n"*/, jobSummary.Params.Creator.UserID, userIdFromPath, p)
				//userIdFromPath = jobSummary.Params.Creator.UserID
				continue
			}

			userItems[quantId] = jobSummary

			viewerGroupId := ""
			if removeIfSharedQuant(jobSummary, sharedItems) {
				viewerGroupId = userGroups["PIXL-FM"]
			}

			if err := migrateQuant(jobSummary, "", coll, userContentBucket, destUserContentBucket, viewerGroupId, fs, dest); err != nil {
				fatalError(err)
			}
		}
	}

	fmt.Printf("Quantification import results: roiID field count: %v, roiIDs field count: %v, elementSetID field count: %v, multiQuants: %v\n",
		roiSetCount, roisSetCount, elementSetSetCount, multiQuantCount)

	fmt.Printf("Quants inserted: %v\n", len(userItems))
	fmt.Println("Adding the following orphaned Quants (shared but original not found):")
	for _, shared := range sharedItems {
		fmt.Printf(" - %v\n", shared.JobID)
		// At this point, it's been shared, so no longer in the OutputFilePath dir that's in the summary. We override it with the real
		// path to the shared item
		srcPath := SrcGetUserQuantPath("shared", shared.Params.DatasetID, "")
		if err := migrateQuant(shared, srcPath, coll, userContentBucket, destUserContentBucket, userGroups["PIXL-FM"], fs, dest); err != nil {
			fatalError(fmt.Errorf("migrateQuant failed for: %v. Error: %v", shared.JobID, err))
		}
	}

	return nil
}

func migrateQuant(jobSummary SrcJobSummaryItem, overrideSrcPath string, coll *mongo.Collection, userContentBucket string, destUserContentBucket string, viewerGroupId string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	var jobStatus protos.JobStatus_Status

	switch jobSummary.Status {
	case SrcJobStarting:
		jobStatus = protos.JobStatus_STARTING
	case SrcJobPreparingNodes:
		jobStatus = protos.JobStatus_PREPARING_NODES
	case SrcJobNodesRunning:
		jobStatus = protos.JobStatus_RUNNING
	case SrcJobGatheringResults:
		jobStatus = protos.JobStatus_GATHERING_RESULTS
	case SrcJobComplete:
		jobStatus = protos.JobStatus_COMPLETE
	case SrcJobError:
		jobStatus = protos.JobStatus_ERROR
	}

	if len(jobSummary.Params.RoiIDs) > 0 && len(jobSummary.Params.RoiID) > 0 && !utils.ItemInSlice(jobSummary.Params.RoiID, jobSummary.Params.RoiIDs) {
		return fmt.Errorf("Both Roi (%v) and Roi IDs is set for quant %v, scan %v, and Roi IDs doesn't contain Roi!", jobSummary.JobID, jobSummary.JobID, jobSummary.Params.DatasetID)
	}

	rois := jobSummary.Params.RoiIDs
	if len(rois) <= 0 && len(jobSummary.Params.RoiID) > 0 {
		rois = []string{jobSummary.Params.RoiID}
	}

	// Write to DB
	destQuant := &protos.QuantificationSummary{
		Id:     jobSummary.JobID,
		ScanId: jobSummary.Params.DatasetID,
		Params: &protos.QuantStartingParameters{
			UserParams: &protos.QuantCreateParams{
				Command: jobSummary.Params.Command,
				Name:    jobSummary.Params.Name,
				ScanId:  jobSummary.Params.DatasetID,
				//Pmcs:           ??,
				Elements:       jobSummary.Params.Elements,
				DetectorConfig: jobSummary.Params.DetectorConfig,
				Parameters:     jobSummary.Params.Parameters,
				RunTimeSec:     uint32(jobSummary.Params.RunTimeSec),
				QuantMode:      jobSummary.Params.QuantMode,
				RoiIDs:         rois,
				IncludeDwells:  jobSummary.Params.IncludeDwells,
			},
			PmcCount:          uint32(jobSummary.Params.PMCCount),
			ScanFilePath:      jobSummary.Params.DatasetPath,
			DataBucket:        jobSummary.Params.DataBucket,
			PiquantJobsBucket: jobSummary.Params.PiquantJobsBucket,
			CoresPerNode:      uint32(jobSummary.Params.CoresPerNode),
			StartUnixTimeSec:  uint32(jobSummary.Params.StartUnixTime),
			RequestorUserId:   utils.FixUserId(jobSummary.Params.Creator.UserID),
			PIQUANTVersion:    jobSummary.Params.PIQUANTVersion,
			Comments:          jobSummary.Params.Comments,
		},
		Elements: jobSummary.Elements,
		Status: &protos.JobStatus{
			JobId:          jobSummary.JobID,
			Status:         jobStatus,
			Message:        jobSummary.Message,
			EndUnixTimeSec: uint32(jobSummary.EndUnixTime),
			OutputFilePath: filepaths.GetQuantPath(utils.FixUserId(jobSummary.Params.Creator.UserID), jobSummary.Params.DatasetID, ""), //jobSummary.OutputFilePath,
			OtherLogFiles:  jobSummary.PiquantLogList,
		},
	}

	_, err := coll.InsertOne(context.TODO(), destQuant)
	if err != nil {
		return err
	}

	err = saveOwnershipItem(jobSummary.JobID, protos.ObjectType_OT_QUANTIFICATION, jobSummary.Params.Creator.UserID, "", viewerGroupId, uint32(jobSummary.EndUnixTime), dest)
	if err != nil {
		return err
	}

	// Save the relevant quantification files to their destination in S3.
	// NOTE: if they are not found, this is an error!
	srcPath := jobSummary.OutputFilePath
	if len(overrideSrcPath) > 0 {
		srcPath = overrideSrcPath
	}
	return saveQuantFiles(jobSummary.Params.DatasetID, jobSummary.JobID, userContentBucket, srcPath, destUserContentBucket, destQuant.Status.OutputFilePath, fs)
}

func removeIfSharedQuant(jobSummary SrcJobSummaryItem, sharedQuantSummaries map[string]SrcJobSummaryItem) bool {
	// Run through all shared quants and see if we find one that matches this one
	for c, sharedItem := range sharedQuantSummaries {
		if jobSummary.Params.DatasetID == sharedItem.Params.DatasetID &&
			jobSummary.Params.Name == sharedItem.Params.Name &&
			jobSummary.Params.QuantMode == sharedItem.Params.QuantMode &&
			jobSummary.Params.Creator.UserID == sharedItem.Params.Creator.UserID &&
			len(jobSummary.Params.Elements) == len(sharedItem.Params.Elements) {
			// Finally, check that the elements array are equal
			for c, elem := range jobSummary.Params.Elements {
				if elem != sharedItem.Params.Elements[c] {
					return false
				}
			}

			// Remove this from the shared list
			delete(sharedQuantSummaries, c)
			return true
		}
	}

	return false
}

const quantificationSubPath = "Quantifications"

// TODO: REMOVE THIS - it's the old path!! Once we no longer need the migration tool it can go
// Retrieves files for a user and dataset ID. If fileName is blank, it only returns the directory path
func SrcGetUserQuantPath(userID string, datasetID string, fileName string) string {
	if len(fileName) > 0 {
		return path.Join(filepaths.RootUserContent, userID, datasetID, quantificationSubPath, fileName)
	}
	return path.Join(filepaths.RootUserContent, userID, datasetID, quantificationSubPath)
}

func saveQuantFiles(datasetId string, quantId string, userContentBucket string, srcPath string, destUserContentBucket string, destPath string, fs fileaccess.FileAccess) error {
	srcPaths := []string{
		path.Join(srcPath, quantId+".bin"),
		path.Join(srcPath, quantId+".csv"),
	}

	dstPaths := []string{
		path.Join(destPath, quantId+".bin"),
		path.Join(destPath, quantId+".csv"),
	}

	failOnError := []bool{
		true,  // Expecting a quant bin file to exist
		false, // CSV is optional
	}

	// Quant log files, we need a listing of these
	s3Path := path.Join(srcPath, filepaths.MakeQuantLogDirName(quantId))
	logFiles, err := fs.ListObjects(userContentBucket, s3Path)
	if err != nil {
		// We just warn here
		log.Printf("Failed to list quant log files at: %v. Skipping...", s3Path)
	} else {
		for c, logPath := range logFiles {
			if quantLogLimitCount > 0 && c >= quantLogLimitCount {
				log.Printf(" Stopping log copy due to quantLogLimitCount = %v", quantLogLimitCount)
				break
			}

			srcPaths = append(srcPaths, logPath)
			dstPaths = append(dstPaths, path.Join(destPath, path.Join(quantId+"-logs", path.Base(logPath))))
			failOnError = append(failOnError, false) // optional
		}
	}

	s3Copy(fs, userContentBucket, srcPaths, destUserContentBucket, dstPaths, failOnError)
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
				fmt.Printf(" SKIPPING import of multi-quant from user: %v aka %v\n", userIdFromPath, usersIdsToIgnore[userIdFromPath])
				continue
			}

			if len(limitToDatasetIds) > 0 && !utils.ItemInSlice(datasetIdFromPath, limitToDatasetIds) {
				fmt.Printf(" SKIPPING multi-quant for dataset id: %v...\n", datasetIdFromPath)
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
