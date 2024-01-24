package main

import (
	"context"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"
	"sync"

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
	SrcJobPreparingNodes   SrcJobStatusValue = "preparing_nodes"
	SrcJobNodesRunning     SrcJobStatusValue = "nodes_running"
	SrcJobGatheringResults SrcJobStatusValue = "gathering_results"
	SrcJobComplete         SrcJobStatusValue = "complete"
	SrcJobError            SrcJobStatusValue = "error"
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

	// Read the summaries
	quantSummaries, err := getQuantSummaryItems(userContentFiles, limitToDatasetIds, userContentBucket, fs)
	if err != nil {
		return err
	}

	// Managed shared vs user ones and match them when possible
	removeDuplicatesWithShared(quantSummaries)

	// Display import stats
	displayImportStats(quantSummaries)

	// Now migrate what's left
	migrateQuantItems(quantSummaries, userContentBucket, destUserContentBucket, userGroups, dest, fs)

	return nil
}

type quantSummaryItem struct {
	quantId         string
	scanId          string
	userIdFromPath  string
	summaryPath     string
	shared          bool
	isOrphanedShare bool
	summaryItem     *SrcJobSummaryItem
}

func getQuantSummaryItems(
	userContentFiles []string,
	limitToDatasetIds []string,
	userContentBucket string,
	fs fileaccess.FileAccess,
) (map[string]quantSummaryItem, error) {
	summariesNeeded := map[string]quantSummaryItem{}

	for _, p := range userContentFiles {
		if strings.Contains(p, "/Quantifications/summary-") {
			// Example: UserContent/<user-id>/<dataset-id>/Quantification/summary-<quant-id>.json

			// If user id is shared, this is a shared quant
			shared := strings.HasPrefix(p, "UserContent/shared/")

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

			// Make sure this is the first time we saw this id
			if _, ok := summariesNeeded[p]; ok {
				return nil, fmt.Errorf("Duplicate quantification ID: %v for path: %v", quantId, p)
			}

			// We'll want this file!
			summariesNeeded[quantId] = quantSummaryItem{
				quantId,
				scanId,
				userIdFromPath,
				p,
				shared,
				shared, // isOrphanedShare - assume they are all orphaned for now
				nil,    // don't have the summary item yet, that's what we'll be reading!
			}
		}
	}

	// Now that we've decided what to read, run some tasks to read them
	summaryReads := make(chan quantSummaryItem, len(summariesNeeded))

	// Read them
	var summaryItemsLock = sync.Mutex{}
	jobSummaryItems := map[string]SrcJobSummaryItem{}

	fmt.Printf("Reading %v job summary files in parallel...\n", len(summariesNeeded))

	var wg sync.WaitGroup

	// Read several at a time
	for w := 0; w < 10; w++ {
		go jobSummaryReadWorker(&wg, summaryReads, &summaryItemsLock, jobSummaryItems, userContentBucket, fs)
	}

	for _, item := range summariesNeeded {
		summaryReads <- item
	}

	close(summaryReads)

	wg.Wait()

	fmt.Printf("Read %v job summary files. Processing...\n", len(summariesNeeded))

	missingSummaryIds := []string{}

	for id, item := range summariesNeeded {
		summary, ok := jobSummaryItems[id]
		if !ok {
			missingSummaryIds = append(missingSummaryIds, id)
		} else {
			item.summaryItem = &summary

			// Overwrite existing
			summariesNeeded[id] = item
		}
	}

	if len(missingSummaryIds) > 0 {
		return nil, fmt.Errorf("Failed to find job summary for id: [%v]", strings.Join(missingSummaryIds, ","))
	}

	return summariesNeeded, nil
}

func removeDuplicatesWithShared(quants map[string]quantSummaryItem) {
	// Find all the user vs shared ones
	userQuants := map[string]quantSummaryItem{}
	sharedQuants := map[string]quantSummaryItem{}
	for id, quant := range quants {
		if quant.shared {
			sharedQuants[id] = quant
		} else {
			userQuants[id] = quant
		}
	}

	// Now run through all user ones and remove them if they are already in shared
	for id, quant := range userQuants {
		sharedId := getSharedIdIfExists(quant, sharedQuants)
		if len(sharedId) > 0 {
			// This user quant is also shared (under a different ID)
			// We delete the user copy and mark the shared one as matched-with-user (this way we can list
			// orphaned shared ones, where user has deleted but shared one exists)
			fmt.Printf("Found %v is user copy of shared %v. Removing user copy and marking shared as not-orphaned.", id, sharedId)

			delete(quants, id)

			sharedQuant, ok := quants[sharedId]
			if !ok {
				fatalError(fmt.Errorf("removeDuplicatesWithShared: Failed to find shared quant: %v", sharedId))
			} else {
				sharedQuant.isOrphanedShare = false
				quants[sharedId] = sharedQuant
			}
		}
	}
}

func displayImportStats(quants map[string]quantSummaryItem) {
	roiSetCount := 0
	roisSetCount := 0
	elementSetSetCount := 0
	multiQuantCount := 0
	sharedCount := 0
	sharedOrphanCount := 0

	for id, quant := range quants {
		jobSummary := quant.summaryItem

		if quant.shared {
			sharedCount++
		}

		if quant.isOrphanedShare {
			if quant.shared {
				sharedOrphanCount++
			} else {
				// sanity check
				fatalError(fmt.Errorf("Found quant %v marked as not shared but orphaned", id))
			}
		}

		// Update counts
		if len(jobSummary.Params.RoiID) > 0 {
			fmt.Printf("%v has RoiID set: %v\n", quant.summaryPath, jobSummary.Params.RoiID)
			roiSetCount++
		}
		if len(jobSummary.Params.RoiIDs) > 0 {
			fmt.Printf("%v has RoiIDs set: [%v]\n", quant.summaryPath, strings.Join(jobSummary.Params.RoiIDs, ","))
			roisSetCount++
		}
		if len(jobSummary.Params.ElementSetID) > 0 {
			elementSetSetCount++
		}
		if jobSummary.Params.QuantMode == SrcQuantModeCombinedMultiQuant || jobSummary.Params.QuantMode == SrcQuantModeABMultiQuant {
			multiQuantCount++
		}

		// Paranoia, make sure both quant IDs match
		if jobSummary.JobID != quant.quantId {
			fatalError(fmt.Errorf("Quant ID mismatch: path had %v, job summary file had %v. Path: %v", quant.quantId, jobSummary.JobID, quant.summaryPath))
		}
		if jobSummary.Params.DatasetID != quant.scanId {
			fatalError(fmt.Errorf("Quant Scan ID mismatch: path had %v, job summary file had %v. Path: %v", quant.scanId, jobSummary.Params.DatasetID, quant.summaryPath))
		}
	}

	fmt.Printf("Quantification import results: roiID field count: %v, roiIDs field count: %v, elementSetID field count: %v, multiQuants: %v\n",
		roiSetCount, roisSetCount, elementSetSetCount, multiQuantCount)

	fmt.Printf("Total quants read: %v, containing %v shared, of which %v are orphaned", len(quants), sharedCount, sharedOrphanCount)
}

func migrateQuantItems(
	quants map[string]quantSummaryItem,
	userContentBucket string,
	destUserContentBucket string,
	userGroups map[string]string,
	dest *mongo.Database,
	fs fileaccess.FileAccess,
) {
	quantMigrationJobs := []quantMigrateJob{}

	fmt.Printf("Migrating %v quants...", len(quants))

	for id, quant := range quants {
		if id != quant.quantId || id != quant.summaryItem.JobID {
			fatalError(fmt.Errorf("migrateQuantItems: quant id mismatch between: stored Id: %v, quantId %v, retrieved job summary id: %v", id, quant.quantId, quant.summaryItem.JobID))
		}

		overrideSrcPath := ""
		if quant.isOrphanedShare {
			// For orphaned ones, we need to look up the path again
			// NOT SURE WHY THIS IS EXACTLY, just following the last bit of code when rewriting... TODO: test me!
			overrideSrcPath = SrcGetUserQuantPath("shared", quant.summaryItem.Params.DatasetID, "")
		}

		viewerGroupId := ""
		if quant.shared {
			viewerGroupId = userGroups["PIXL-FM"]
		}

		quantMigrationJobs = append(quantMigrationJobs, quantMigrateJob{
			quant, overrideSrcPath, userContentBucket, destUserContentBucket, viewerGroupId,
		})
	}

	var wg sync.WaitGroup

	// Start a pool of quant migrators
	jobs := make(chan quantMigrateJob, len(quantMigrationJobs))
	for w := 0; w < 4; w++ {
		go migrateQuantWorker(&wg, jobs, fs, dest)
	}

	// Add jobs!
	for _, job := range quantMigrationJobs {
		jobs <- job
	}

	close(jobs)

	wg.Wait()

	fmt.Printf("Quant migration of %v quants is complete!\n", len(quants))
}

func jobSummaryReadWorker(wg *sync.WaitGroup, summaries <-chan quantSummaryItem, summaryItemsLock *sync.Mutex, jobSummaryItems map[string]SrcJobSummaryItem, userContentBucket string, fs fileaccess.FileAccess) {
	defer wg.Done()
	wg.Add(1)

	for s := range summaries {
		jobSummary := SrcJobSummaryItem{}
		err := fs.ReadJSON(userContentBucket, s.summaryPath, &jobSummary, false)
		if err != nil {
			fatalError(err)
		}

		// Save it in the waiting struct
		//defer summaryItemsLock.Unlock()
		summaryItemsLock.Lock()

		// Ensure this doesn't already exist
		if _, ok := jobSummaryItems[s.quantId]; ok {
			fatalError(fmt.Errorf("jobSummaryReadWorker found %v already loaded", s.quantId))
		}

		jobSummaryItems[s.quantId] = jobSummary
		summaryItemsLock.Unlock()

		fmt.Printf("Summary (remaining %v): %v read OK\n", len(summaries), s.summaryPath)
	}
}

type quantMigrateJob struct {
	quant                 quantSummaryItem
	overrideSrcPath       string
	userContentBucket     string
	destUserContentBucket string
	viewerGroupId         string
}

func migrateQuantWorker(wg *sync.WaitGroup, jobs <-chan quantMigrateJob, fs fileaccess.FileAccess, dest *mongo.Database) {
	defer wg.Done()
	wg.Add(1)

	for j := range jobs {
		id := addImportTask(fmt.Sprintf("migrateQuant datasetID: %v, quantID: %v, jobOutputPath: %v", j.quant.scanId, j.quant.quantId, j.quant.summaryItem.OutputFilePath))
		err := migrateQuant(*j.quant.summaryItem, j.overrideSrcPath, j.userContentBucket, j.destUserContentBucket, j.viewerGroupId, fs, dest)
		finishImportTask(id, err)

		fmt.Printf("Migrated (remaining: %v) quant id: %v.\n", len(jobs), j.quant.quantId)
	}
}

func migrateQuant(jobSummary SrcJobSummaryItem, overrideSrcPath string, userContentBucket string, destUserContentBucket string, viewerGroupId string, fs fileaccess.FileAccess, dest *mongo.Database) error {
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

	coll := dest.Collection(dbCollections.QuantificationsName)
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

func getSharedIdIfExists(userQuant quantSummaryItem, sharedQuantSummaries map[string]quantSummaryItem) string {
	// Run through all shared quants and see if we find one that matches this one
	for id, sharedItem := range sharedQuantSummaries {
		if userQuant.summaryItem.Params.DatasetID == sharedItem.summaryItem.Params.DatasetID &&
			userQuant.summaryItem.Params.Name == sharedItem.summaryItem.Params.Name &&
			userQuant.summaryItem.Params.QuantMode == sharedItem.summaryItem.Params.QuantMode &&
			userQuant.summaryItem.Params.Creator.UserID == sharedItem.summaryItem.Params.Creator.UserID &&
			len(userQuant.summaryItem.Params.Elements) == len(sharedItem.summaryItem.Params.Elements) &&
			utils.SlicesEqual(userQuant.summaryItem.Params.Elements, sharedItem.summaryItem.Params.Elements) {
			return id
		}
	}

	return ""
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
