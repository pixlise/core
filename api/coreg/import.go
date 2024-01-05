package coreg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/job"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func StartCoregImport(triggerUrl string, hctx wsHelpers.HandlerContext) (string, error) {
	if len(triggerUrl) <= 0 {
		return "", errorwithstatus.MakeBadRequestError(errors.New("MarsViewerExport trigger Url is empty"))
	}

	i := coregUpdater{hctx}

	// Start an image coreg import job (this is a Lambda function)
	// Once it completes, we have the data we need, so we can treat it as a "normal" image importing task
	jobStatus, err := job.AddJob("coreg", uint32(hctx.Svcs.Config.ImportJobMaxTimeSec), hctx.Svcs.MongoDB, hctx.Svcs.IDGen, hctx.Svcs.TimeStamper, hctx.Svcs.Log, i.sendUpdate)
	jobId := ""
	if jobStatus != nil {
		jobId = jobStatus.JobId
	}

	if err != nil || len(jobId) < 0 {
		returnErr := fmt.Errorf("Failed to add job watcher for coreg import Job ID: %v. Error was: %v", jobId, err)
		hctx.Svcs.Log.Errorf("%v", returnErr)
		return "", returnErr
	}

	// We can now trigger the lambda
	// NOTE: here we build the same structure that triggered us, but we exclude the points data so we don't exceed
	// the SQS 256kb limit. The lambda doesn't care about the points anyway, only we do once the lambda has completed!
	coregReq := CoregJobRequest{jobId, hctx.Svcs.Config.EnvironmentName, triggerUrl}
	msg, err := json.Marshal(coregReq)
	if err != nil {
		returnErr := fmt.Errorf("Failed to create coreg job trigger message for job ID: %v", jobId)
		job.CompleteJob(jobId, false, returnErr.Error(), "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return "", returnErr
	}

	_, err = hctx.Svcs.SQS.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(msg)),
		QueueUrl:    aws.String(hctx.Svcs.Config.CoregSqsQueueUrl),
	})

	if err != nil {
		returnErr := fmt.Errorf("Failed to trigger coreg job. ID: %v. Error: %v", jobId, err)
		job.CompleteJob(jobId, false, returnErr.Error(), "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return "", returnErr
	}

	return jobId, nil
}

type coregUpdater struct {
	hctx wsHelpers.HandlerContext
}

func (i *coregUpdater) sendUpdate(status *protos.JobStatus) {
	// NOTE: The coreg image import job sets state GATHERING_RESULTS when it has downloaded everything
	// so here we trigger off that to do our part, after which we can mark the job as COMPLETE or ERROR
	if status.Status == protos.JobStatus_GATHERING_RESULTS {
		// NOTE: If this fails, it will set the job status to ERROR and we'll
		// get another call to update...
		completeMarsViewerImportJob(status.JobId, i.hctx)
		return
	}

	wsUpd := protos.WSMessage{
		Contents: &protos.WSMessage_ImportMarsViewerImageUpd{
			ImportMarsViewerImageUpd: &protos.ImportMarsViewerImageUpd{
				Status: status,
			},
		},
	}

	wsHelpers.SendForSession(i.hctx.Session, &wsUpd)
}

// Should be called after Coreg Import Lambda has completed successfully
func completeMarsViewerImportJob(jobId string, hctx wsHelpers.HandlerContext) {
	// Read the job completion entry from DB
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.CoregJobCollection)
	dbResult := coll.FindOne(ctx, bson.M{"_id": jobId}, options.FindOne())
	if dbResult.Err() != nil {
		failJob(fmt.Sprintf("Failed to find Coreg Job completion record for: %v. Error: %v", jobId, dbResult.Err()), jobId, hctx)
		return
	}

	coregResult := CoregJobResult{}
	err := dbResult.Decode(&coregResult)
	if err != nil {
		failJob(fmt.Sprintf("Failed to decode Coreg Job completion record for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	// At this point we should have everything ready to go - our own bucket should contain all images
	// and we have the mars viewer export msg containing any points we require so lets import the warped images we received!
	// Firstly, read the export from MV
	marsViewerExport := protos.MarsViewerExport{}
	mvBucket, err := fileaccess.GetBucketFromS3Url(coregResult.MarsViewerExportUrl)
	if err != nil {
		failJob(fmt.Sprintf("Failed to read Coreg Job files for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	mvPath, err := fileaccess.GetPathFromS3Url(coregResult.MarsViewerExportUrl)
	if err != nil {
		failJob(fmt.Sprintf("Failed to read Coreg Job files for: %v. Error: %v", jobId, err), jobId, hctx)
		return
	}

	err = hctx.Svcs.FS.ReadJSON(mvBucket, mvPath, &marsViewerExport, false)
	if err != nil {
		msg := fmt.Sprintf("Failed to read MarsViewer export json for: %v. Error: %v", jobId, err)
		job.CompleteJob(jobId, false, msg, "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return
	}

	// Loop through each warped image, read it, along with beam locations (in observation for it)
	for _, warpedImg := range marsViewerExport.WarpedOverlayImages {
		warpedFileName := path.Base(warpedImg.WarpedImageUrl)

		// Find this in our bucket
		ourCopyPath := ""
		for _, ourWarpedImgPath := range coregResult.WarpedImageUrls {
			if strings.HasSuffix(ourWarpedImgPath.NewUri, warpedFileName) {
				// We found it!
				ourCopyPath = ourWarpedImgPath.NewUri
				break
			}
		}

		if len(ourCopyPath) <= 0 {
			failJob(fmt.Sprintf("Failed to read warped image: %v for import job: %v", warpedFileName, jobId), jobId, hctx)
			return
		}

		// Grab any observations for it (?)

		// Import this image into db along with beam locations, and image into S3
	}

	job.CompleteJob(jobId, true, "Import complete", "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
}

func failJob(errMsg string, jobId string, hctx wsHelpers.HandlerContext) {
	job.CompleteJob(jobId, false, errMsg, "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
}
