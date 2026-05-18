package jobmanager

import (
	jobexecutor "github.com/pixlise/core/v4/api/job/executor"
)

func (jm *JobManager) ListJobs() ([]jobexecutor.JobGroupConfig, error) {
	return nil, nil
}

func (jm *JobManager) GetJob(JobId string) (jobexecutor.JobGroupConfig, error) {
	return jobexecutor.JobGroupConfig{}, nil
}
