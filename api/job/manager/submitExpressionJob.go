package jobmanager

import (
	"fmt"
	"path"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/filepaths"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/job/jobnode"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func (jm *JobManager) SubmitExpressionJob(scanId, quantId, expressionId, roiId, memoCacheKey string, requestorUserSess *sessionuser.SessionUser, requestorSession *melody.Session) (*protos.JobStatus, error) {
	// Call the internal one, log the resulting errors if any
	status, err := jm.internalSubmitExpressionJob(scanId, quantId, expressionId, roiId, memoCacheKey, requestorUserSess, requestorSession)
	if err != nil {
		jm.svcs.Log.Errorf("SubmitExpressionJob error: %v", err)
	}
	return status, err
}

func (jm *JobManager) internalSubmitExpressionJob(scanId, quantId, expressionId, roiId, memoCacheKey string, requestorUserSess *sessionuser.SessionUser, requestorSession *melody.Session) (*protos.JobStatus, error) {
	// If we don't have a user, use the built-in PIXLISE user
	requestorUserId := sessionuser.PIXLISESystemUserId
	if requestorUserSess != nil {
		requestorUserId = requestorUserSess.User.Id
	}

	// Generate a job ID
	jobId := fmt.Sprintf("expr-lua-%v", jm.svcs.IDGen.GenObjectID())

	jobS3Path := filepaths.GetJobDataPath(scanId, jobId, "")

	source, expr, err := expressionrunner.FetchSourceCode(expressionId, scanId, quantId, requestorUserId, jm.svcs)
	if err != nil {
		return nil, err
	}

	// Upload source file and make list of required files for job to execute
	requiredFiles := []jobconfig.JobFilePath{}

	sourceFileName := "source.lua"
	remoteSourcePath := filepaths.GetJobDataPath(scanId, jobId, sourceFileName)
	err = jm.svcs.FS.WriteObject(jm.svcs.Config.PiquantJobsBucket, remoteSourcePath, []byte(source))
	if err != nil {
		return nil, err
	}

	requiredFiles = append(requiredFiles, jobconfig.JobFilePath{
		LocalPath:    sourceFileName,
		RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
		RemotePath:   remoteSourcePath,
	})

	// Read and validate scan
	scanItem, err := scan.ReadScanItem(scanId, jm.svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// TODO: Potentially put in paths for quant/scan files to downlink, but we shouldn't need this
	// because the expression runner code knows how to download the files from their source dataset bucket
	/*

		// Read and validate quantification
		filter := bson.M{"_id": quantId}
		opts := options.FindOne()
		quantResult := jm.svcs.MongoDB.Collection(dbCollections.QuantificationsName).FindOne(context.TODO(), filter, opts)

		if quantResult.Err() != nil {
			return quantResult.Err()
		}

		quant := &protos.QuantificationSummary{}
		err := quantResult.Decode(quant)
		if err != nil {
			return err
		}
	*/

	jg := &jobconfig.JobGroupConfig{
		JobGroupId:       jobId,
		JobType:          protos.JobType_JT_RUN_EXPRESSION,
		CompletionMethod: JobComplete_LuaExpression,
		DockerImage:      jm.svcs.Config.Jobs.RunnerDockerImage,
		NodeCount:        1,
		NodeConfig: jobconfig.JobConfig{
			JobId:         jobId + "-node",
			RequiredFiles: requiredFiles,
			Command:       jobnode.LuaExpressionCommand, //"lua5.3",
			Args:          []string{"scanId=" + scanId, "quantId=" + quantId, "expressionId=" + expressionId, "memoKey=" + memoCacheKey},
			OutputFiles: []jobconfig.JobFilePath{
				{
					LocalPath:    "stdout",
					RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
					RemotePath:   path.Join(jobS3Path, "output", "stdout.log"),
				},
				{
					LocalPath:    jobnode.ExpressionJobOutputFileName,
					RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
					RemotePath:   path.Join(jobS3Path, "output", jobnode.ExpressionJobOutputFileName),
				},
			},
		},
		AssociatedScanId: scanId,
		JobName:          fmt.Sprintf("%v-%v", scanItem.Title, expr.Name),
		//ElementList
		RequestorUserId: requestorUserId,
	}

	return jm.internalSubmitJob(jg, requestorSession)
}
