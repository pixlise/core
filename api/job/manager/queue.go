package jobmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/core/singleinstance"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

// Submit function for each kind of job type we support
func (jm *JobManager) QueueJob(jg *jobconfig.JobGroupConfig) error {
	nowUnixSec := jm.svcs.TimeStamper.GetTimeNowSec()

	qItems := []interface{}{}

	for c := uint(0); c < jg.NodeCount; c++ {
		cfg := jg.NodeConfig.FlattenJobConfig(c)

		qItems = append(qItems, &protos.JobQueueItem{
			JobId:                       cfg.JobId,
			JobGroupId:                  jg.JobGroupId,
			AssociatedScanId:            jg.AssociatedScanId,
			NodeIndex:                   uint32(c),
			CreatedTimeStampUnixSec:     nowUnixSec,
			LastUpdatedTimeStampUnixSec: nowUnixSec,
			State:                       protos.JobQueueItem_UNKNOWN,
		})
	}

	ctx := context.TODO()

	// Set both states in one transaction
	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := jm.svcs.MongoDB.Client().StartSession()
	if err != nil {
		return err
	}
	defer sess.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// Insert job queue items
		coll := jm.svcs.MongoDB.Collection(dbCollections.JobQueueName)

		result, err := coll.InsertMany(ctx, qItems)
		if err != nil {
			return nil, err
		}

		if len(result.InsertedIDs) != int(jg.NodeCount) {
			jm.svcs.Log.Errorf("Unexpected result count %v after inserting %v job queue items for job group: %v", len(result.InsertedIDs), int(jg.NodeCount), jg.JobGroupId)
		}

		// Update job state
		err = jm.updateJobStatus(jg.JobGroupId, protos.JobStatus_PREPARING_NODES, fmt.Sprintf("Preparing %v nodes...", len(qItems)), true)
		return nil, err
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	return err
}

// Internal function to check if there are any outstanding jobs on startup
func (jm *JobManager) startupCheckQueue(startupQueueCheckDelaySec int) {
	// Wait a little so we don't do this instantly on startup
	time.Sleep(time.Duration(startupQueueCheckDelaySec) * time.Second)
	jm.runCheckJobQueueOnce("jobmanager-start")
}

var RATE_LIMIT_SEC = uint(2)

func (jm *JobManager) listenToJobQueue() bool {
	return job.ListenToJobQueue([]string{"insert", "update"}, jm.svcs.MongoDB, jm.svcs.TimeStamper, jm.svcs.Log, RATE_LIMIT_SEC, jm.onJobQueueChanged)
}

func (jm *JobManager) onJobQueueChanged(jobItem *protos.JobQueueItem) {

	// Here we ignore the queue item, we want to check the entire queue to find job groups
	// that have finished or whatever, so run that check here
	jm.runCheckJobQueueOnce("jobmanager-queue")
}

func (jm *JobManager) runCheckJobQueueOnce(sourceId string) {
	// In case there are multiple APIs running, we here have to decide who is going to do the check
	// so we only check jobs once (avoiding duplicate starts)
	err := singleinstance.HandleOnce(sourceId, jm.svcs.InstanceId, func(sourceId string) {
		// Read all items and work out what
		//jm.svcs.Log.Infof("HandleOnce id %v, instance %v...", sourceId, jm.svcs.InstanceId)
		err := jm.checkJobQueue()
		if err != nil {
			jm.svcs.Log.Errorf("checkJobQueue (HandleOnce id %v, instance %v) failed: %v", sourceId, jm.svcs.InstanceId, err)
		}
	}, jm.svcs.MongoDB, jm.svcs.TimeStamper, jm.svcs.Log)

	if err != nil {
		jm.svcs.Log.Errorf("Failed to HandleOnce id %v, instance %v. Error: %v", sourceId, jm.svcs.InstanceId, err)
	}
}

// This is run (by one EC2 API instance) at startup or when we are told there is a change in the job queue.
// The intent is to find jobs:
// - That have completed: Run a completion task (if needed) and remove them from the queue
// - Detect jobs that have failed (because a sub-job has failed) and fail the whole job group, remove from queue
// - Perhaps detect jobs that have timed out and mark them as failed
// - Detect if we have enough JobNodes, if not, start more so the jobs can start getting picked up and processed

// A similar function will be run on a JobNode which is likely running on one or more separate EC2 instances
// which has a different purpose:
//   - Detect new not-yet-run jobs and claim them, run them locally if there is spare capacity in terms of docker
//     containers to CPU core ratio
func (jm *JobManager) checkJobQueue() error {
	groupsAndJobs, err := job.ReadJobQueue(jm.svcs.MongoDB)
	if err != nil {
		return err
	}

	runningInstanceIds, err := jm.getRunningNodes()
	if err != nil {
		return err
	}

	jm.svcs.Log.Debugf("CheckJobQueue found %v job groups", len(groupsAndJobs))

	// Check if we have any jobs that have timed out - eg if it was started on a job node and that node is no longer
	// active... or somehow it didn't finish this is where we want to eventually clean it up
	nowUnixSec := jm.svcs.TimeStamper.GetTimeNowSec()
	for _, jobs := range groupsAndJobs {
		for _, jobItem := range jobs {
			err = jm.checkJobTimeout(jobItem, runningInstanceIds, nowUnixSec)
			if err != nil {
				return err
			}
		}
	}

	existingJobStates := map[string]*protos.JobStatus{}
	for jobGroupId := range groupsAndJobs {
		s, err := jm.readJobStatus(jobGroupId)
		if err != nil {
			jm.svcs.Log.Errorf("CheckJobQueue job group %v not found!", jobGroupId)
		} else {
			existingJobStates[jobGroupId] = s
		}
	}

	// Main queue processing:
	// - Remove any that have all completed
	// - Mark job group as running if a child node is running
	// - Get a list of ALL jobs in queue that are not yet running
	notStartedIds := []string{}
	for jobGroupId, jobs := range groupsAndJobs {
		assignedCount, runningCount, completedCount, failedCount, idsNotStarted := jm.countJobNodeStates(jobs)
		ranCount := failedCount + completedCount

		notStartedIds = append(notStartedIds, idsNotStarted...)

		existingJobStatus := existingJobStates[jobGroupId]

		jm.svcs.Log.Debugf("  CheckJobQueue job group %v has %v ran, %v completed nodes of %v", jobGroupId, ranCount, completedCount, len(jobs))

		// If the job group has any jobs in the queue marked running, mark the job group as running too
		if runningCount > 0 {
			runningStatus := fmt.Sprintf("Running on %v nodes with %v waiting, %v assigned, %v complete, %v failed", runningCount, len(idsNotStarted), assignedCount, completedCount, failedCount)
			if existingJobStatus != nil && existingJobStatus.Message != runningStatus {
				jm.updateJobStatusWithInMemory(jobGroupId, protos.JobStatus_RUNNING, runningStatus, existingJobStatus, true, "CheckJobQueue RunningCheck")
			}
		} else {
			// None are in a running state...

			// If they've all been completed, do the completion task (if there is one)
			if completedCount >= len(jobs) && failedCount == 0 {
				// We only try to complete a job if we have a status for it!
				if existingJobStatus != nil {
					jm.completeJob(jobGroupId, len(jobs), existingJobStatus)
				}
			} // NOT else!! The following code needs to be able to run in complete AND failed scenarios!

			// If they've all been run, delete it from the queue
			if completedCount+failedCount >= len(jobs) {
				jm.clearJob(jobGroupId, groupsAndJobs)

				// If they're not all completed, we just mark the job as failed
				if failedCount > 0 {
					if err = jm.updateJobStatusWithInMemory(jobGroupId, protos.JobStatus_ERROR, fmt.Sprintf("%v nodes failed", failedCount), existingJobStatus, true, "CheckJobQueue FailCheck"); err == nil {
						jm.svcs.Log.Infof("  Marking job %v as ERROR due to nodes not all completing", jobGroupId)
					}
				}
			}
		}
	}

	// Start nodes as needed, assigning jobs to each one
	jm.svcs.Log.Debugf("  CheckJobQueue found %v not-started jobs", len(notStartedIds))
	if len(notStartedIds) > 0 && jm.startNodes {
		return jm.startJobNodes(notStartedIds)
	}

	return nil
}

// Counts how many of each state we find for the queued jobs:
// ASSIGNED state jobs
// RUNNING state jobs
// COMPLETE state jobs
// FAILED state jobs
// NotStartedIds: JobIds of jobs that are in the UNKNOWN state
func (jm *JobManager) countJobNodeStates(jobs []*protos.JobQueueItem) (int, int, int, int, []string) {
	assigned := 0
	running := 0
	completed := 0
	failed := 0
	notStartedIds := []string{}

	for _, job := range jobs {
		if job.State == protos.JobQueueItem_ASSIGNED {
			assigned = assigned + 1
		}

		if job.State == protos.JobQueueItem_RUNNING {
			running = running + 1
		}

		if job.State == protos.JobQueueItem_COMPLETE {
			completed = completed + 1
		}

		if job.State == protos.JobQueueItem_FAILED {
			failed = failed + 1
		}

		if job.State == protos.JobQueueItem_UNKNOWN {
			notStartedIds = append(notStartedIds, job.JobId)
		}
	}

	return assigned, running, completed, failed, notStartedIds
}

func (jm *JobManager) checkJobTimeout(jobItem *protos.JobQueueItem, runningInstanceIds []string, nowUnixSec int64) error {
	if jobItem.State != protos.JobQueueItem_ASSIGNED && jobItem.State != protos.JobQueueItem_RUNNING {
		// Job cannot have timed out in its current state
		return nil
	}

	// If the instance is no longer with us, or if a long time has passed, we drop the job
	secSinceUpdate := nowUnixSec - jobItem.LastUpdatedTimeStampUnixSec

	isNodeGone := jobItem.State == protos.JobQueueItem_RUNNING && !utils.ItemInSlice(jobItem.InstanceId, runningInstanceIds)
	if !isNodeGone && secSinceUpdate < int64(jm.svcs.Config.JobMaxNodeRunTimeSec) {
		// Job is not yet dead/timed out
		return nil
	}

	// Mark this job node as failed
	jobItem.Message = "Node did not complete job."
	if jobItem.State == protos.JobQueueItem_ASSIGNED {
		jobItem.Message = "Node did not start job."
	}
	if isNodeGone {
		jobItem.Message = fmt.Sprintf("Node timed out after %v seconds.", secSinceUpdate)
	}

	jobItem.State = protos.JobQueueItem_FAILED

	jm.svcs.Log.Debugf("  CheckJobQueue detected timed out/incomplete job: %v. Message: %v", jobItem.JobId, jobItem.Message)

	// Write it out
	err := job.UpdateJobQueueItem(jobItem.JobId, jobItem.State, jobItem.Message, jobItem.JobGroupId, "", jm.svcs.MongoDB, jm.svcs.TimeStamper)
	if err != nil {
		return fmt.Errorf("JobManager queue check failed to mark job %v as timed out. Error: %v", jobItem.JobId, err)
	}

	// At this point we would want to mark the job group's status as failed but now that we've marked it as failed here it'll get picked up and
	// the job group marked as failed elsewhere
	return nil
}

func (jm *JobManager) completeJob(jobGroupId string, nodeCount int, existingStatus *protos.JobStatus) {
	if existingStatus.Status >= protos.JobStatus_GATHERING_RESULTS {
		jm.svcs.Log.Errorf("Skipped job completion for for: %v - its status is %v", jobGroupId, existingStatus.Status)
		return
	}

	jm.svcs.Log.Debugf("  CheckJobQueue running job group %v completion task...", jobGroupId)
	// Set the job status to gathering results
	jm.updateJobStatusWithInMemory(jobGroupId, protos.JobStatus_GATHERING_RESULTS, fmt.Sprintf("Combining CSVs from %v nodes...", nodeCount), existingStatus, true, "CheckJobQueue completeJob")

	err := jm.onJobGroupCompletion(jobGroupId, existingStatus)
	if err != nil {
		// Set the job status to gathering results
		jm.updateJobStatusWithInMemory(jobGroupId, protos.JobStatus_ERROR, fmt.Sprintf("Failed to complete job group %v: %v", jobGroupId, err), existingStatus, true, "CheckJobQueue completeJob-failed")
	} else {
		// Set the job status to gathering results
		if err = jm.updateJobStatusWithInMemory(jobGroupId, protos.JobStatus_COMPLETE, fmt.Sprintf("Nodes ran: %v", nodeCount), existingStatus, true, "CheckJobQueue completeJob-success"); err == nil {
			jm.svcs.Log.Debugf("  CheckJobQueue completed job group %v", jobGroupId)
		}
	}
}

func (jm *JobManager) updateJobStatusWithInMemory(jobId string, status protos.JobStatus_Status, message string, inMemoryStatus *protos.JobStatus, updateClient bool, action string) error {
	err := jm.updateJobStatus(jobId, status, message, updateClient)
	if err != nil {
		jm.svcs.Log.Errorf("  %v error: %v", err)
	} else {
		inMemoryStatus.Status = status
		inMemoryStatus.Message = message
	}
	return err
}

// Clears a job from the job queue (in-memory and DB)
func (jm *JobManager) clearJob(jobGroupId string, groupsAndJobs map[string][]*protos.JobQueueItem) {
	jm.svcs.Log.Debugf("  CheckJobQueue clearing job queue items for %v", jobGroupId)

	// Clear in-memory
	delete(groupsAndJobs, jobGroupId)

	// Delete from DB
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobQueueName)

	delResult, err := coll.DeleteMany(ctx, bson.M{"jobgroupid": jobGroupId})
	if err != nil {
		jm.svcs.Log.Errorf("Failed to delete completed jobs for group: %v. %v", jobGroupId, err)
	} else if delResult.DeletedCount <= 0 {
		jm.svcs.Log.Errorf("Unexpected delete count %v after deleting completed jobs for group: %v", delResult.DeletedCount, jobGroupId)
	}
}

func (jm *JobManager) onJobGroupCompletion(jobGroupId string, jobStatus *protos.JobStatus) error {
	// Check if we have to do anything
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobsName)

	filter := bson.M{"_id": jobGroupId}
	jobGroup := coll.FindOne(ctx, filter, options.FindOne())
	if jobGroup.Err() != nil {
		return jobGroup.Err()
	}

	jg := &jobconfig.JobGroupConfig{}
	if err := jobGroup.Decode(jg); err != nil {
		return err
	}

	// Check if we have this completion method registered at all
	if len(jg.CompletionMethod) <= 0 {
		jm.svcs.Log.Infof("Job Group %v has no completion method defined", jobGroupId)
		return nil
	}

	completionMethod, ok := jm.jobCompletionMethods[jg.CompletionMethod]

	if !ok {
		return fmt.Errorf("Job completion failed, method %v unknown", jg.CompletionMethod)
	}

	// Run the method - get the session if we can find it
	return completionMethod(jg, jobStatus, jm.userSessionLookup[jg.RequestorUserId], jm.svcs)
}
