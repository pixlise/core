package jobmanager

import (
	"github.com/olahol/melody"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/job/jobnode"
	"github.com/pixlise/core/v4/api/services"
	protos "github.com/pixlise/core/v4/generated-protos"
)

var JobComplete_CombineCSVs = "combine_csvs"
var JobComplete_SingleCSV = "single_csvs"

type JobManagerCompletionFunction func(*jobconfig.JobGroupConfig, *protos.JobStatus, *melody.Session, *services.APIServices) error

type JobManager struct {
	svcs                 *services.APIServices
	jobCompletionMethods map[string]JobManagerCompletionFunction
	useFileCache         bool
	localJobNode         *jobnode.JobNode
	startNodes           bool
	startedNodeCount     uint
	userSessionLookup    map[string]*melody.Session
}

func CreateJobManager(svcs *services.APIServices, startupQueueCheckDelaySec int, monitorJobQueue bool, useFileCache bool, startNodes bool) (*JobManager, error) {
	// Make a job manager
	jm := &JobManager{
		svcs: svcs,
		jobCompletionMethods: map[string]JobManagerCompletionFunction{
			JobComplete_CombineCSVs: completeQuantMultiNodeJob,
			JobComplete_SingleCSV:   completeQuantSingleMapJob,
		},
		useFileCache:      useFileCache,
		startNodes:        startNodes,
		userSessionLookup: map[string]*melody.Session{},
	}

	if startupQueueCheckDelaySec > 0 {
		// Check for unfinished jobs in a little bit, and start monitoring the job queue for new insertions
		go jm.startupCheckQueue(startupQueueCheckDelaySec)
	}

	if monitorJobQueue {
		// Do we want to listen to the job queue? Realistically always, but for testing it can be disabled
		// if we're also not interested in the startup queue checks...
		go jm.listenToJobQueue()
	}

	return jm, nil
}

func (jm *JobManager) RegisterCompletionMethod(name string, f JobManagerCompletionFunction) {
	jm.jobCompletionMethods[name] = f
}
