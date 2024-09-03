package wsHandler

import (
	"errors"
	"path"
	"strings"

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
	i := quantification.MakeQuantJobUpdater(req.Params, hctx.Session, hctx.Svcs.Notifier, hctx.Svcs.MongoDB, hctx.Svcs.FS, hctx.Svcs.Config.UsersBucket)

	updater := i.SendQuantJobUpdate
	if req.Params.Command != "map" {
		updater = i.SendEphemeralQuantJobUpdate
	}

	status, err := quantification.CreateJob(req.Params, hctx.SessUser.User.Id, hctx.Svcs, &hctx.SessUser, updater)

	if err != nil {
		return nil, err
	}

	// Just pass back the generated job status, updates will happen via the update function passed in
	return &protos.QuantCreateResp{Status: status}, nil
}
