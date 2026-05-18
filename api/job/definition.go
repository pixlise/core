package job

import (
	"fmt"

	"github.com/pixlise/core/v4/core/utils"
)

type NodeIndexMethod int

const (
	NodeIndexMethod_None = iota
	NodeIndexMethod_Local
	NodeIndexMethod_Remote
	NodeIndexMethod_Both
)

type JobFilePath struct {
	// The remote file info
	RemoteBucket string
	RemotePath   string

	// Local copy
	LocalPath string

	// Apply node index to this path (or not)
	// If this is false, we just use the path as is, but if
	// it's set to true, the local and remote paths have the
	// node number inserted in between the file name and its
	// extension. Eg node.pmcs becomes node00001.pmcs
	ApplyNodeIndex NodeIndexMethod
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

	// eg if this contains a 2, it means
	// we need to apply the node index to
	// Args[2] before running this job
	ArgIndexToApplyNodeIndexes []int

	// What to upload on completion (if file doesn't exist, it can be ignored with a warning)
	OutputFiles []JobFilePath
}

func (c JobConfig) FlattenJobConfig(nodeIndex uint) JobConfig {
	newCfg := JobConfig{
		JobId:         fmt.Sprintf("%v-%v", c.JobId, nodeIndex),
		RequiredFiles: []JobFilePath{},
		Command:       c.Command,
		Args:          make([]string, len(c.Args)),
		OutputFiles:   []JobFilePath{},

		// No need to supply this really... we're applying indexes to the args as needed right now
		//ArgIndexToApplyNodeIndexes: c.ArgIndexToApplyNodeIndexes,
	}

	for i, arg := range c.Args {
		// If we have any arguments marked as needing the node index applied, apply it here
		if utils.ItemInSlice(i, c.ArgIndexToApplyNodeIndexes) {
			newCfg.Args[i] = utils.ApplyIndexToFileName(arg, nodeIndex, true)
		} else {
			newCfg.Args[i] = arg
		}
	}

	for _, f := range c.RequiredFiles {
		newCfg.RequiredFiles = append(newCfg.RequiredFiles, JobFilePath{
			ApplyNodeIndex: f.ApplyNodeIndex,
			RemoteBucket:   f.RemoteBucket,
			RemotePath:     utils.ApplyIndexToFileName(f.RemotePath, nodeIndex, f.ApplyNodeIndex == NodeIndexMethod_Both || f.ApplyNodeIndex == NodeIndexMethod_Remote),
			LocalPath:      utils.ApplyIndexToFileName(f.LocalPath, nodeIndex, f.ApplyNodeIndex == NodeIndexMethod_Both || f.ApplyNodeIndex == NodeIndexMethod_Local),
		})
	}

	for _, f := range c.OutputFiles {
		newCfg.OutputFiles = append(newCfg.OutputFiles, JobFilePath{
			ApplyNodeIndex: f.ApplyNodeIndex,
			RemoteBucket:   f.RemoteBucket,
			RemotePath:     utils.ApplyIndexToFileName(f.RemotePath, nodeIndex, f.ApplyNodeIndex == NodeIndexMethod_Both || f.ApplyNodeIndex == NodeIndexMethod_Remote),
			LocalPath:      utils.ApplyIndexToFileName(f.LocalPath, nodeIndex, f.ApplyNodeIndex == NodeIndexMethod_Both || f.ApplyNodeIndex == NodeIndexMethod_Local),
		})
	}

	return newCfg
}

var JobConfigEnvVar = "JOB_CONFIG"
