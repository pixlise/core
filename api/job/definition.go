package job

type JobFilePath struct {
	// The remote file info
	RemoteBucket string
	RemotePath   string

	// Local copy
	LocalPath string
}

type JobConfig struct {
	// The job id
	JobId string

	// Logging method - If these are empty, we just log to stdout
	// LogCloudwatchGroup string
	// LogCloudwatchStream string

	// What files are required to be present when running the job?
	RequiredFiles []JobFilePath

	// What command to execute
	Command string
	Args    []string

	// What to upload on completion (if file doesn't exist, it can be ignored with a warning)
	OutputFiles []JobFilePath
}

func (c JobConfig) Copy() JobConfig {
	newCfg := JobConfig{
		JobId:         c.JobId,
		RequiredFiles: []JobFilePath{},
		Command:       c.Command,
		Args:          c.Args,
		OutputFiles:   []JobFilePath{},
	}

	for _, f := range c.RequiredFiles {
		newCfg.RequiredFiles = append(newCfg.RequiredFiles, JobFilePath{
			RemoteBucket: f.RemoteBucket,
			RemotePath:   f.RemotePath,
			LocalPath:    f.LocalPath,
		})
	}

	for _, f := range c.OutputFiles {
		newCfg.OutputFiles = append(newCfg.OutputFiles, JobFilePath{
			RemoteBucket: f.RemoteBucket,
			RemotePath:   f.RemotePath,
			LocalPath:    f.LocalPath,
		})
	}

	return newCfg
}

var JobConfigEnvVar = "JOB_CONFIG"
