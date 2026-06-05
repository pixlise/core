package jobmanager

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
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
			CompletionMethod: "NonExistantMethod",
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

	// Read DB stuff back, ensuring it's in an expected state

	cur, err := svcs.MongoDB.Collection(dbCollections.JobsName).Find(ctx, bson.M{})
	fmt.Printf("read jobs: %v\n", err)
	if cur != nil {
		jobs := []*jobconfig.JobGroupConfig{}
		err = cur.All(context.TODO(), &jobs)
		if err != nil {
			log.Fatalf("read jobs: %v\n", err)
		}
		for _, j := range jobs {
			fmt.Printf("%v|%v\n", j.JobGroupId, j.JobName)
		}
	}

	cur, err = svcs.MongoDB.Collection(dbCollections.JobStatusName).Find(ctx, bson.M{})
	fmt.Printf("read job statuses: %v\n", err)
	if cur != nil {
		statuses := []*protos.JobStatus{}
		err = cur.All(context.TODO(), &statuses)
		if err != nil {
			log.Fatalf("read jobs: %v\n", err)
		}
		for _, s := range statuses {
			fmt.Printf("%v|%v|%v\n", s.JobId, s.Status, s.Message)
		}
	}

	cur, err = svcs.MongoDB.Collection(dbCollections.JobQueueName).Find(ctx, bson.M{})
	fmt.Printf("read job queue: %v\n", err)
	if cur != nil {
		q := []*protos.JobQueueItem{}
		err = cur.All(context.TODO(), &q)
		if err != nil {
			log.Fatalf("read jobs: %v\n", err)
		}
		for _, item := range q {
			fmt.Printf("%v|%v|%v\n", item.JobId, item.State, item.Message)
		}
	}

	// Output:
	// create: <nil>, instance: the-test-instance
	// insert jobs: <nil>
	// insert job statuses: <nil>
	// insert queues: <nil>
	// completeJob func called for: quant-id999, state: GATHERING_RESULTS, session exists: false
	// read jobs: <nil>
	// quant-id123|job1
	// quant-id456|job2
	// quant-id789|job3
	// quant-id998|job3
	// quant-id999|job4
	// quant-id007|job5
	// read job statuses: <nil>
	// quant-id123|RUNNING|
	// quant-id456|RUNNING|
	// quant-id789|ERROR|1 nodes failed
	// quant-id998|ERROR|Failed to complete job group quant-id998: Job completion failed, method NonExistantMethod unknown
	// quant-id999|COMPLETE|Nodes ran: 2
	// quant-id007|ERROR|1 nodes failed
	// read job queue: <nil>
	// quant-id123-node-0|UNKNOWN|
	// quant-id123-node-1|UNKNOWN|
	// quant-id456-node-0|UNKNOWN|
	// quant-id456-node-1|COMPLETE|
}

func completeJobFunc(jg *jobconfig.JobGroupConfig, jstatus *protos.JobStatus, sess *melody.Session, svcs *services.APIServices) error {
	fmt.Printf("completeJob func called for: %v, state: %v, session exists: %v\n", jg.JobGroupId, jstatus.Status, sess != nil)
	return nil
}
