package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/job"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/encoding/protojson"
)

func HandleImportMarsViewerImageReq(req *protos.ImportMarsViewerImageReq, hctx wsHelpers.HandlerContext) (*protos.ImportMarsViewerImageResp, error) {
	// Check for valid-looking request
	if req.MarsViewerExport == nil ||
		len(req.MarsViewerExport.BaseImageUrl) <= 0 ||
		len(req.MarsViewerExport.MarsviewerLink) <= 0 ||
		len(req.MarsViewerExport.Observations) <= 0 ||
		len(req.MarsViewerExport.WarpedOverlayImages) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("MarsViewerExport struct missing/missing fields"))
	}

	i := coregUpdater{
		hctx,
		req.MarsViewerExport,
	}

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
		return nil, returnErr
	}

	// We can now trigger the lambda
	// NOTE: here we build the same structure that triggered us, but we exclude the points data so we don't exceed
	// the SQS 256kb limit. The lambda doesn't care about the points anyway, only we do once the lambda has completed!
	coregReq := &protos.ImageCoregImportJob{
		JobId: jobId,
		Params: &protos.MarsViewerExport{
			BaseImageUrl:        req.MarsViewerExport.BaseImageUrl,
			MarsviewerLink:      req.MarsViewerExport.MarsviewerLink,
			WarpedOverlayImages: req.MarsViewerExport.WarpedOverlayImages,
			Observations:        []*protos.MVObservation{},
		},
	}

	for _, obs := range req.MarsViewerExport.Observations {
		coregReq.Params.Observations = append(coregReq.Params.Observations, &protos.MVObservation{
			Interpolated:           obs.Interpolated,
			ObservationCSVFilename: obs.ObservationCSVFilename,
			ContextImageUrl:        obs.ContextImageUrl,
			Site:                   obs.Site,
			Drive:                  obs.Drive,
		})
	}

	msg, err := protojson.Marshal(coregReq)
	if err != nil {
		returnErr := fmt.Errorf("Failed to create coreg job trigger message for job ID: %v", jobId)
		job.CompleteJob(jobId, false, returnErr.Error(), "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return nil, returnErr
	}

	hctx.Svcs.SQS.SendMessage(&sqs.SendMessageInput{
		MessageBody: aws.String(string(msg)),
		QueueUrl:    aws.String(hctx.Svcs.Config.CoregSqsQueueUrl),
	})

	return &protos.ImportMarsViewerImageResp{
		JobId: jobId,
	}, nil
}

type coregUpdater struct {
	hctx  wsHelpers.HandlerContext
	mvReq *protos.MarsViewerExport
}

func (i *coregUpdater) sendUpdate(status *protos.JobStatus) {
	// NOTE: The coreg image import job sets state GATHERING_RESULTS when it has downloaded everything
	// so here we trigger off that to do our part, after which we can mark the job as COMPLETE or ERROR
	if status.Status == protos.JobStatus_GATHERING_RESULTS {
		// NOTE: If this fails, it will set the job status to ERROR and we'll
		// get another call to update...
		completeMarsViewerImportJob(status.JobId, i.mvReq)
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

type JobOutput struct{} // <-- Define in API, pull into Lambda as go module

// Should be called after Coreg Import Lambda has completed successfully
func completeMarsViewerImportJob(jobId string, marsViewerReq *protos.MarsViewerExport, hctx wsHelpers.HandlerContext) {
	// Read the job completion entry from DB
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.CoregJobCollection)
	dbResult := coll.FindOne(ctx, bson.M{"_id": jobId}, options.FindOne())
	if dbResult.Err() != nil {
		msg := fmt.Sprintf("Failed to find Coreg Job completion record for: %v. Error: %v", jobId, dbResult.Err())
		job.CompleteJob(jobId, false, msg, "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return
	}

	coregResult := JobOutput{}
	err := dbResult.Decode(&coregResult)
	if err != nil {
		msg := fmt.Sprintf("Failed to decode Coreg Job completion record for: %v. Error: %v", jobId, err)
		job.CompleteJob(jobId, false, msg, "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
		return
	}

	// At this point we should have everything ready to go - our own bucket should contain all images
	// and we have the mars viewer export msg containing any points we require so lets import this image!

	job.CompleteJob(jobId, true, "Import complete", "", []string{}, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper, hctx.Svcs.Log)
}
