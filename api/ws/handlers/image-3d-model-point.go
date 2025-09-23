package wsHandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/gdsfilename"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleImage3DModelPointsReq(req *protos.Image3DModelPointsReq, hctx wsHelpers.HandlerContext) (*protos.Image3DModelPointsResp, error) {
	// We want to store the image name sans version (applicable to MCC image names mainly!)
	imageReadName := dataImportHelpers.GetImageNameSansVersion(req.ImageName)

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.Image3DPointsName)
	imgFound := coll.FindOne(ctx, bson.M{"_id": imageReadName}, options.FindOne())
	if imgFound.Err() != nil {
		if imgFound.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("3D points not found for image: \"%v\"", req.ImageName))
		}
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to read 3d points for image \"%v\": %v", req.ImageName, imgFound.Err()))
	}

	pts := &protos.Image3DPoints{}
	err := imgFound.Decode(pts)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to decode 3d points for image \"%v\": %v", req.ImageName, err))
	}

	return &protos.Image3DModelPointsResp{
		Points: pts,
	}, nil
}

func HandleImage3DModelPointUploadReq(req *protos.Image3DModelPointUploadReq, hctx wsHelpers.HandlerContext) (*protos.Image3DModelPointUploadResp, error) {
	if req.Points == nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Points is empty"))
	}

	if len(req.Points.ImageName) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Points.ImageName is not set"))
	}

	if len(req.Points.Points) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Point list is empty"))
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.ImagesName)
	imagePath := req.Points.ImageName
	imgFound := coll.FindOne(ctx, bson.M{"_id": imagePath}, options.FindOne())
	if imgFound.Err() != nil {
		if imgFound.Err() == mongo.ErrNoDocuments {
			// Check, maybe the path part is missing
			found := false

			if !strings.Contains(imagePath, "/") {
				// If it's a PDS style file name, we can get the RTT and prepend it and look up again
				meta, err := gdsfilename.ParseFileName(imagePath)
				if err == nil {
					rtt, err := meta.RTT()

					if err == nil {
						imagePath = fmt.Sprintf("%v/%v", rtt, imagePath)

						imgFound = coll.FindOne(ctx, bson.M{"_id": imagePath}, options.FindOne())
						if imgFound.Err() == nil {
							found = true
						}
					}
				}
			}
			if !found {
				return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Image \"%v\" not found", imagePath))
			}
		} else {
			return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to check image \"%v\": %v", imagePath, imgFound.Err()))
		}
	}

	// Request is valid, image exists, so lets store this
	// We want to store the image name sans version (applicable to MCC image names mainly!)
	imageStoreName := dataImportHelpers.GetImageNameSansVersion(imagePath)
	coll = hctx.Svcs.MongoDB.Collection(dbCollections.Image3DPointsName)

	opt := options.Update().SetUpsert(true)

	req.Points.ImageName = imageStoreName
	result, err := coll.UpdateByID(ctx, imageStoreName, bson.D{{Key: "$set", Value: req.Points}}, opt)
	if err != nil {
		return nil, err
	}

	if result.UpsertedCount == 0 && result.ModifiedCount == 0 {
		hctx.Svcs.Log.Errorf("HandleImage3DModelPointUploadReq got unexpected upsert result: %+v", result)
	}

	return &protos.Image3DModelPointUploadResp{}, nil
}
