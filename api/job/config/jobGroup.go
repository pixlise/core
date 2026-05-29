package jobconfig

import (
	"github.com/pixlise/core/v4/api/quantification"
	protos "github.com/pixlise/core/v4/generated-protos"
)

type JobGroupConfig struct {
	JobGroupId       string         `bson:"_id,omitempty"` // Job group ID
	JobType          protos.JobType // Job type, mostly for annotation of job state
	CompletionMethod string         // What to do when the job is completed, one of the JobComplete_* names
	DockerImage      string         // Docker image to run in each node
	FastStart        bool           // May go unused - but could be a way to run it locally on this machine if we know it's a quick job
	NodeCount        uint           // Node count, because NodeConfig can be asked to retrieve config of each node, but here we know the total
	NodeConfig       JobConfig      // Node config sources

	// Job meta-data - this contains task specific fields
	// TODO: Remove this or make it generic somehow! Perhaps move these to a json file that is saved with the job and expect it to be available at the end?
	// The problem is they are available at job creation, but also needed in the job completion task for certain jobs
	AssociatedScanId string   // Empty if none, or if it's across scans
	JobName          string   // Optional job name, eg used for quants
	ElementList      []string // Optional element list, eg used for quants
	RequestorUserId  string
	OutputTitle      string // Optional, ends up in the title of an output file, eg the first row of a CSV
	Combined         bool
	QuantByROI       bool
	ROIs             []quantification.ROIItemWithPMCs
	// Need configs for:
	// NodeOutputCombining - how to combine the outputs, eg PIQUANT map commands
	// Do we need to write overall job output/logs somewhere?
}
