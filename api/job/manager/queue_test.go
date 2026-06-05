package jobmanager

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func Example_jobmanager_QueueStartup() {
	timestamps := []int64{}

	// queue time stamp
	for c := int64(0); c < 30; c++ {
		timestamps = append(timestamps, int64(1668142582)+c)
	}

	origWD, _, svcs := initJobManagerTest(nil, timestamps)
	defer os.Chdir(origWD)

	svcs.Config.NodeCountOverride = 4
	svcs.Config.JobMaxNodeRunTimeSec = 1800
	svcs.Log = &logger.StdOutLogger{}
	svcs.Log.SetLogLevel(logger.LogError)

	jm, err := CreateJobManager(&svcs, 0, false, false, false)
	fmt.Printf("create: %v, instance: %v\n", err, jm.svcs.InstanceId)

	jm.RegisterCompletionMethod("completeJob", completeJobFunc)

	// Add some items to the queue
	ctx := context.TODO()

	jobs := []interface{}{}
	jobs = append(jobs,
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id123",
			JobType:          protos.JobType_JT_RUN_FIT,
			CompletionMethod: "",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "",
			JobName:          "job1",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id456",
			JobType:          protos.JobType_JT_IMPORT_SCAN,
			CompletionMethod: "",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "1234567890",
			JobName:          "job2",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id789",
			JobType:          protos.JobType_JT_RUN_QUANT,
			CompletionMethod: "NonExistantJob",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "",
			JobName:          "job3",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id998",
			JobType:          protos.JobType_JT_RUN_QUANT,
			CompletionMethod: "NonExistantJob",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "",
			JobName:          "job3",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id999",
			JobType:          protos.JobType_JT_RUN_QUANT,
			CompletionMethod: "completeJob",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "",
			JobName:          "job4",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
		&jobconfig.JobGroupConfig{
			JobGroupId:       "quant-id007",
			JobType:          protos.JobType_JT_RUN_QUANT,
			CompletionMethod: "completeJob",
			DockerImage:      "job-container",
			FastStart:        false,
			NodeCount:        2,
			NodeConfig:       jobconfig.JobConfig{},
			AssociatedScanId: "",
			JobName:          "job5",
			ElementList:      []string{},
			RequestorUserId:  "abc123",
		},
	)

	jobStatuses := []interface{}{}
	jobStatuses = append(jobStatuses,
		&protos.JobStatus{
			JobId:           "quant-id123",
			JobType:         protos.JobType_JT_RUN_FIT,
			Status:          protos.JobStatus_RUNNING,
			RequestorUserId: "abc123",
		},
		&protos.JobStatus{
			JobId:           "quant-id456",
			JobType:         protos.JobType_JT_IMPORT_SCAN,
			Status:          protos.JobStatus_RUNNING,
			RequestorUserId: "abc123",
		},
		&protos.JobStatus{
			JobId:           "quant-id789",
			JobType:         protos.JobType_JT_RUN_QUANT,
			Status:          protos.JobStatus_RUNNING,
			RequestorUserId: "abc123",
		},
		&protos.JobStatus{
			JobId:           "quant-id998",
			JobType:         protos.JobType_JT_RUN_QUANT,
			Status:          protos.JobStatus_RUNNING,
			RequestorUserId: "abc123",
		},
		&protos.JobStatus{
			JobId:           "quant-id999",
			JobType:         protos.JobType_JT_RUN_QUANT,
			Status:          protos.JobStatus_RUNNING,
			RequestorUserId: "abc123",
		},
		&protos.JobStatus{
			JobId:           "quant-id007",
			JobType:         protos.JobType_JT_RUN_QUANT,
			Status:          protos.JobStatus_STARTING,
			RequestorUserId: "abc123",
		},
	)

	jobQItems := []interface{}{}
	jobQItems = append(jobQItems,
		// Job with nothing finished
		&protos.JobQueueItem{
			JobId:                       "quant-id123-node-0",
			JobGroupId:                  "quant-id123",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_UNKNOWN,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id123-node-1",
			JobGroupId:                  "quant-id123",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_UNKNOWN,
		},
		// Job with some finished
		&protos.JobQueueItem{
			JobId:                       "quant-id456-node-0",
			JobGroupId:                  "quant-id456",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_UNKNOWN,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id456-node-1",
			JobGroupId:                  "quant-id456",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		// Job with all finished
		&protos.JobQueueItem{
			JobId:                       "quant-id789-node-0",
			JobGroupId:                  "quant-id789",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id789-node-1",
			JobGroupId:                  "quant-id789",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_FAILED,
		},
		// Job with all completed
		&protos.JobQueueItem{
			JobId:                       "quant-id998-node-0",
			JobGroupId:                  "quant-id998",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id998-node-1",
			JobGroupId:                  "quant-id998",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		// Job with all completed
		&protos.JobQueueItem{
			JobId:                       "quant-id999-node-0",
			JobGroupId:                  "quant-id999",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id999-node-1",
			JobGroupId:                  "quant-id999",
			AssociatedScanId:            "scan1",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668142581,
			LastUpdatedTimeStampUnixSec: 1668142581,
			State:                       protos.JobQueueItem_COMPLETE,
		},
		&protos.JobQueueItem{
			JobId:                       "quant-id007-node-1",
			JobGroupId:                  "quant-id007",
			AssociatedScanId:            "scan3",
			NodeIndex:                   1,
			CreatedTimeStampUnixSec:     1668112585,
			LastUpdatedTimeStampUnixSec: 1668112586,
			State:                       protos.JobQueueItem_ASSIGNED,
		},
	)
	//go jm.listenToJobQueue()

	_, err = svcs.MongoDB.Collection(dbCollections.JobsName).InsertMany(ctx, jobs)
	fmt.Printf("insert jobs: %v\n", err)

	_, err = svcs.MongoDB.Collection(dbCollections.JobStatusName).InsertMany(ctx, jobStatuses)
	fmt.Printf("insert job statuses: %v\n", err)

	_, err = svcs.MongoDB.Collection(dbCollections.JobQueueName).InsertMany(ctx, jobQItems)
	fmt.Printf("insert queues: %v\n", err)

	jm.startupCheckQueue(1)

	time.Sleep(3 * time.Second)

	// Output:
	// create: <nil>, instance: the-test-instance
	// insert jobs: <nil>
	// insert job statuses: <nil>
	// insert queues: <nil>
	// completeJob func called for: quant-id999
}

func completeJobFunc(jg *jobconfig.JobGroupConfig, jstatus *protos.JobStatus, svcs *services.APIServices) error {
	fmt.Printf("completeJob func called for: %v\n", jg.JobGroupId)
	return nil
}
