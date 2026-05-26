package jobmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/core/singleinstance"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobQueueName)

	result, err := coll.InsertMany(ctx, qItems)
	if err != nil {
		return err
	}

	if len(result.InsertedIDs) != int(jg.NodeCount) {
		jm.svcs.Log.Errorf("Unexpected result count %v after inserting %v job queue items for job group: %v", len(result.InsertedIDs), int(jg.NodeCount), jg.JobGroupId)
	}

	return nil
}

// Internal function to check if there are any outstanding jobs on startup
func (jm *JobManager) startupCheckQueue(startupQueueCheckDelaySec int) {
	// Wait a little so we don't do this instantly on startup
	time.Sleep(time.Duration(startupQueueCheckDelaySec) * time.Second)
	jm.runCheckJobQueueOnce("jobmanager-start")
}

func (jm *JobManager) listenToJobQueue() {
	job.ListenToJobQueue([]string{"insert", "update"}, jm.svcs.MongoDB, jm.svcs.Log, jm.onJobQueueChanged)
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
		jm.svcs.Log.Infof("HandleOnce id %v, instance %v...", sourceId, jm.svcs.InstanceId)
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

	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobQueueName)

	// Remove any that have all completed
	for jobGroupId, jobs := range groupsAndJobs {
		ranCount := 0
		completed := 0
		for _, job := range jobs {
			if job.State == protos.JobQueueItem_COMPLETE {
				completed = completed + 1
			}

			if job.State == protos.JobQueueItem_COMPLETE || job.State == protos.JobQueueItem_FAILED {
				ranCount = ranCount + 1
			}
		}

		// If they've all been completed, do the completion task (if there is one)
		if completed >= len(jobs) {
			existingStatus, err := jm.readJobStatus(jobGroupId)
			if err != nil {
				jm.svcs.Log.Errorf("Failed to read existing job group status for: %v. %v", jobGroupId, err)
			} else {
				// We only try to complete a job if we have a status for it!
				if existingStatus.Status < protos.JobStatus_GATHERING_RESULTS {
					// Set the job status to gathering results
					updatedStatus, _ := jm.updateJobStatus(jobGroupId, protos.JobStatus_GATHERING_RESULTS, fmt.Sprintf("Combining CSVs from %v nodes...", len(jobs)), "", existingStatus)

					err = jm.onJobGroupCompletion(jobGroupId, updatedStatus)
					if err != nil {
						jobErr := fmt.Errorf("Failed to complete job group %v: %v", jobGroupId, err)

						// Set the job status to gathering results
						jm.updateJobStatus(jobGroupId, protos.JobStatus_ERROR, fmt.Sprintf("%v", jobErr), "", existingStatus)
					} else {
						// Set the job status to gathering results
						jm.updateJobStatus(jobGroupId, protos.JobStatus_COMPLETE, fmt.Sprintf("Nodes ran: %v", len(jobs)), "", existingStatus)
					}
				} else {
					jm.svcs.Log.Errorf("Skipped job completion for for: %v - its status is %v", jobGroupId, existingStatus.Status)
				}
			}
		}

		// If they've all been run, delete it
		if ranCount >= len(jobs) {
			delete(groupsAndJobs, jobGroupId)
			delResult, err := coll.DeleteMany(ctx, bson.M{"jobgroupid": jobGroupId})
			if err != nil {
				jm.svcs.Log.Errorf("Failed to delete completed jobs for group: %v. %v", jobGroupId, err)
			} else if delResult.DeletedCount <= 0 {
				jm.svcs.Log.Errorf("Unexpected delete count %v after deleting completed jobs for group: %v", delResult.DeletedCount, jobGroupId)
			}
		}
	}

	// NOTE: Job starting doesn't happen here, it happens in the JobNode listener

	return nil
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

	// Run the method
	return completionMethod(jg, jobStatus, jm.svcs)
}
