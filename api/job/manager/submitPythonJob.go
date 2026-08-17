package jobmanager

import (
	"fmt"
	"path"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job/jobnode"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	"github.com/pixlise/core/v4/api/sessionuser"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func (jm *JobManager) SubmitPythonJob(repoId, scriptName, scanId, quantId string, requestorUserSess *sessionuser.SessionUser, requestorSession *melody.Session) (*protos.JobStatus, error) {
	// Call the internal one, log the resulting errors if any
	status, err := jm.internalSubmitPythonJob(repoId, scriptName, scanId, quantId, requestorUserSess, requestorSession)
	if err != nil {
		jm.svcs.Log.Errorf("SubmitQuantJob error: %v", err)
	}
	return status, err
}

func (jm *JobManager) internalSubmitPythonJob(repoId, scriptName, scanId, quantId string, requestorUserSess *sessionuser.SessionUser, requestorSession *melody.Session) (*protos.JobStatus, error) {
	// Check that the repo exists
	repo := &protos.SourceRepository{}
	if err := expressionrunner.ReadOne(dbCollections.SourceRepositoriesName, bson.M{"_id": repoId}, repo, jm.svcs.MongoDB); err != nil {
		return nil, fmt.Errorf("Failed to read repository details for %v", err)
	}

	// If we don't have a user, use the built-in PIXLISE user
	requestorUserId := sessionuser.PIXLISESystemUserId
	if requestorUserSess != nil {
		requestorUserId = requestorUserSess.User.Id
	}

	// Generate a job ID
	jobId := fmt.Sprintf("python-%v", jm.svcs.IDGen.GenObjectID())
	jobS3Path := filepaths.GetJobDataPath(scanId, jobId, "")

	requiredFiles := []*protos.JobFilePath{}

	jg := &protos.JobGroupConfig{
		JobGroupId:       jobId,
		JobType:          protos.JobType_JT_RUN_PYTHON_SCRIPT,
		CompletionMethod: JobComplete_PythonScript,
		DockerImage:      jm.svcs.Config.Jobs.RunnerDockerImage,
		NodeCount:        1,
		NodeConfig: &protos.JobConfig{
			JobId: jobId + "-node",

			RequiredFiles: requiredFiles,

			Command: jobnode.PythonCommand,
			Args: []string{
				"scanId=" + scanId,
				"quantId=" + quantId,
				"scriptName=" + scriptName,
				"repoId=" + repo.Id,
			},

			OutputFiles: []*protos.JobFilePath{
				{
					LocalPath:    "stdout",
					RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
					RemotePath:   path.Join(jobS3Path, "output", "stdout.log"),
				},
			},
		},
		AssociatedScanId: scanId,
		JobName:          scriptName,
		RequestorUserId:  requestorUserId,
	}

	return jm.internalSubmitJob(jg, requestorSession)
}
