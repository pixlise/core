package wsHandler

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleJobListReq(req *protos.JobListReq, hctx wsHelpers.HandlerContext) (*protos.JobListResp, error) {
	// Work out if requestor is an admin or a normal user
	isAdmin := hctx.SessUser.Permissions["PIXLISE_ADMIN"]

	/*filter, _, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_QUANTIFICATION, hctx)
	if err != nil {
		return nil, err
	}*/
	filter := bson.M{}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.JobStatusName)
	opts := options.Find()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.JobStatus{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	itemsToSend := []*protos.JobStatus{}
	if isAdmin {
		itemsToSend = items
	} else {
		// Find only the jobs that were requested by this user
		for _, item := range items {
			if item.RequestorUserId == hctx.SessUser.User.Id {
				itemsToSend = append(itemsToSend, item)
			}
		}
	}

	return &protos.JobListResp{
		Jobs: itemsToSend,
	}, nil
}
