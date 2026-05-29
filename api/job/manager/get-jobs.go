package jobmanager

import jobconfig "github.com/pixlise/core/v4/api/job/config"

func (jm *JobManager) ListJobs() ([]jobconfig.JobGroupConfig, error) {
	return nil, nil
}

func (jm *JobManager) GetJob(JobId string) (jobconfig.JobGroupConfig, error) {
	return jobconfig.JobGroupConfig{}, nil
}
