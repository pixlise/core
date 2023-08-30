package wsHandler

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/piquant"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandlePiquantConfigListReq(req *protos.PiquantConfigListReq, hctx wsHelpers.HandlerContext) (*protos.PiquantConfigListResp, error) {
	// Return a list of all piquant configs we have stored
	// TODO: Handle paging... this could eventually be > 1000 files, but that's a while away!
	paths, err := hctx.Svcs.FS.ListObjects(hctx.Svcs.Config.ConfigBucket, filepaths.RootDetectorConfig+"/")
	if err != nil {
		hctx.Svcs.Log.Errorf("Failed to list piquant configs in %v/%v: %v", hctx.Svcs.Config.ConfigBucket, filepaths.RootDetectorConfig, err)
		return nil, err
	}

	// Return the names of the configs (dir names)
	configNamesFiltered := map[string]bool{}
	for _, path := range paths {
		bits := strings.Split(path, "/")
		if len(bits) > 2 {
			configNamesFiltered[bits[1]] = true
		}
	}

	// Form a list
	names := utils.GetMapKeys(configNamesFiltered)
	sort.Strings(names)

	return &protos.PiquantConfigListResp{ConfigNames: names}, err
}

func HandlePiquantConfigVersionsListReq(req *protos.PiquantConfigVersionsListReq, hctx wsHelpers.HandlerContext) (*protos.PiquantConfigVersionsListResp, error) {
	if err := wsHelpers.CheckStringField(&req.ConfigId, "ConfigId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Get a list of PIQUANT config versions too
	versions := piquant.GetPiquantConfigVersions(hctx.Svcs, req.ConfigId)

	return &protos.PiquantConfigVersionsListResp{
		Versions: versions,
	}, nil
}

func HandlePiquantConfigVersionReq(req *protos.PiquantConfigVersionReq, hctx wsHelpers.HandlerContext) (*protos.PiquantConfigVersionResp, error) {
	if err := wsHelpers.CheckStringField(&req.Version, "Version", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.ConfigId, "ConfigId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	cfg, err := piquant.GetPIQUANTConfig(hctx.Svcs, req.ConfigId, req.Version)
	if err != nil {
		return nil, err
	}

	return &protos.PiquantConfigVersionResp{
		PiquantConfig: cfg,
	}, nil
}

// TODO: need to query versions from github container registry or something similar???
func HandlePiquantVersionListReq(req *protos.PiquantVersionListReq, hctx wsHelpers.HandlerContext) (*protos.PiquantVersionListResp, error) {
	return nil, errors.New("HandlePiquantVersionListReq not implemented yet")
}

func HandlePiquantCurrentVersionReq(req *protos.PiquantCurrentVersionReq, hctx wsHelpers.HandlerContext) (*protos.PiquantCurrentVersionResp, error) {
	// Look up the PIQUANT version currently set
	result := hctx.Svcs.MongoDB.Collection(dbCollections.PiquantVersionName).FindOne(context.TODO(), bson.M{"_id": "current"})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError("PIQUANT version")
		}
		return nil, result.Err()
	}

	ver := protos.PiquantVersion{}
	err := result.Decode(&ver)
	if err != nil {
		return nil, err
	}

	return &protos.PiquantCurrentVersionResp{PiquantVersion: &ver}, nil
}

func HandlePiquantWriteCurrentVersionReq(req *protos.PiquantWriteCurrentVersionReq, hctx wsHelpers.HandlerContext) (*protos.PiquantWriteCurrentVersionResp, error) {
	if err := wsHelpers.CheckStringField(&req.PiquantVersion, "PiquantVersion", 1, 100); err != nil {
		return nil, err
	}

	// Overwrite the current PIQUANT version
	update := bson.D{
		bson.E{Key: "version", Value: req.PiquantVersion},
		bson.E{Key: "modifiedunixsec", Value: hctx.Svcs.TimeStamper.GetTimeNowSec()},
		bson.E{Key: "modifieruserid", Value: hctx.SessUser.User.Id},
	}
	opts := options.Update().SetUpsert(true)

	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.PiquantVersionName).UpdateByID(context.TODO(), "current", bson.D{{Key: "$set", Value: update}}, opts)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("PiquantWriteCurrentVersionReq UpdateByID result had unexpected counts %+v", result)
	}

	return &protos.PiquantWriteCurrentVersionResp{}, nil
}
