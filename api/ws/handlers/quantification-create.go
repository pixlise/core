package wsHandler

import (
	"errors"
	"path"
	"strings"
	"sync"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleQuantCreateReq(req *protos.QuantCreateReq, hctx wsHelpers.HandlerContext) (*protos.QuantCreateResp, error) {
	err := quantification.IsValidCreateParam(req.Params, hctx)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// At this point, we're assuming that the detector config is a valid config name / version. We need this to be the path of the config in S3
	// so here we convert it and ensure it's valid
	detectorConfigBits := strings.Split(req.Params.DetectorConfig, "/")
	if len(detectorConfigBits) != 2 || len(detectorConfigBits[0]) < 0 || len(detectorConfigBits[1]) < 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("DetectorConfig not in expected format"))
	}

	// Form the string
	// NOTE: we would want to use this:
	// req.DetectorConfig = filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], "")
	// But can't because then the root "/DetectorConfig" is added twice!
	req.Params.DetectorConfig = path.Join(detectorConfigBits[0], filepaths.PiquantConfigSubDir, detectorConfigBits[1])

	// Run the quantification job
	var wg sync.WaitGroup

	i := quantJobUpdater{
		hctx.Session,
		hctx.Melody,
	}

	status, err := quantification.CreateJob(req.Params, hctx.SessUser.User.Id, hctx, &wg, i.sendQuantJobUpdate)

	if err != nil {
		return nil, err
	}

	// If it's NOT a map command, we wait around for the result and pass it back in the response
	// but for map commands, we just pass back the generated job status
	if req.Params.Command == "map" {
		return &protos.QuantCreateResp{Status: status}, nil
	}

	// Wait around for the output file to appear, or for the job to end up in an error state
	wg.Wait()

	// Return error or the resulting CSV, whichever happened
	userOutputFilePath := filepaths.GetUserLastPiquantOutputPath(hctx.SessUser.User.Id, req.Params.ScanId, req.Params.Command, filepaths.QuantLastOutputFileName+".csv")
	bytes, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.UsersBucket, userOutputFilePath)
	if err != nil {
		return nil, errors.New("PIQUANT command: " + req.Params.Command + " failed.")
	}

	return &protos.QuantCreateResp{ResultData: bytes}, nil
}

type quantJobUpdater struct {
	session *melody.Session
	melody  *melody.Melody
}

func (i *quantJobUpdater) sendQuantJobUpdate(status *protos.JobStatus) {
	wsUpd := protos.WSMessage{
		Contents: &protos.WSMessage_QuantCreateUpd{
			QuantCreateUpd: &protos.QuantCreateUpd{
				Status: status,
			},
		},
	}

	wsHelpers.SendForSession(i.session, &wsUpd)
}
