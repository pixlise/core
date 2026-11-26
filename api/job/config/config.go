package jobconfig

import (
	"fmt"

	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func FlattenJobConfig(c *protos.JobConfig, nodeIndex uint) *protos.JobConfig {
	newCfg := &protos.JobConfig{
		JobId:         fmt.Sprintf("%v-%v", c.JobId, nodeIndex),
		RequiredFiles: []*protos.JobFilePath{},
		Command:       c.Command,
		Args:          make([]string, len(c.Args)),
		OutputFiles:   []*protos.JobFilePath{},

		// No need to supply this really... we're applying indexes to the args as needed right now
		//ArgIndexToApplyNodeIndexes: c.ArgIndexToApplyNodeIndexes,
	}

	for i, arg := range c.Args {
		// If we have any arguments marked as needing the node index applied, apply it here
		if utils.ItemInSlice(int32(i), c.ArgIndexToApplyNodeIndexes) {
			newCfg.Args[i] = utils.ApplyIndexToFileName(arg, nodeIndex, true)
		} else {
			newCfg.Args[i] = arg
		}
	}

	for _, f := range c.RequiredFiles {
		newCfg.RequiredFiles = append(newCfg.RequiredFiles, &protos.JobFilePath{
			ApplyNodeIndex: f.ApplyNodeIndex,
			RemoteBucket:   f.RemoteBucket,
			RemotePath:     utils.ApplyIndexToFileName(f.RemotePath, nodeIndex, f.ApplyNodeIndex == protos.NodeIndexMethod_BOTH || f.ApplyNodeIndex == protos.NodeIndexMethod_REMOTE),
			LocalPath:      utils.ApplyIndexToFileName(f.LocalPath, nodeIndex, f.ApplyNodeIndex == protos.NodeIndexMethod_BOTH || f.ApplyNodeIndex == protos.NodeIndexMethod_LOCAL),
		})
	}

	for _, f := range c.OutputFiles {
		newCfg.OutputFiles = append(newCfg.OutputFiles, &protos.JobFilePath{
			ApplyNodeIndex: f.ApplyNodeIndex,
			RemoteBucket:   f.RemoteBucket,
			RemotePath:     utils.ApplyIndexToFileName(f.RemotePath, nodeIndex, f.ApplyNodeIndex == protos.NodeIndexMethod_BOTH || f.ApplyNodeIndex == protos.NodeIndexMethod_REMOTE),
			LocalPath:      utils.ApplyIndexToFileName(f.LocalPath, nodeIndex, f.ApplyNodeIndex == protos.NodeIndexMethod_BOTH || f.ApplyNodeIndex == protos.NodeIndexMethod_LOCAL),
		})
	}

	return newCfg
}
