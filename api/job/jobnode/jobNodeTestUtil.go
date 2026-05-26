package jobnode

import (
	"log"

	"github.com/pixlise/core/v4/api/job"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Public for tests ONLY!
func (jn *JobNode) OnNewJobQueueItemForTest() {
	groupsAndJobs, err := job.ReadJobQueue(jn.db)

	if err != nil {
		log.Fatalf("OnNewJobQueueItem %v", err)
	}

	for _, jobs := range groupsAndJobs {
		for _, jobItem := range jobs {
			if jobItem.State == protos.JobQueueItem_UNKNOWN {
				jn.onNewJobQueueItem(jobItem)
				return
			}
		}
	}
}
