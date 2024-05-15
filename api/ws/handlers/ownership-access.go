package wsHandler

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleGetOwnershipReq(req *protos.GetOwnershipReq, hctx wsHelpers.HandlerContext) ([]*protos.GetOwnershipResp, error) {
	if err := wsHelpers.CheckStringField(&req.ObjectId, "ObjectId", 1, wsHelpers.IdFieldMaxLength*2 /* Tests have longer ids anyway... */); err != nil {
		return nil, err
	}

	owner, err := wsHelpers.CheckObjectAccess(false, req.ObjectId, req.ObjectType, hctx)
	if err != nil {
		return nil, err
	}

	return []*protos.GetOwnershipResp{&protos.GetOwnershipResp{
		Ownership: owner,
	}}, nil
}

func readToMap(ids []string, theMap *map[string]bool) {
	if ids == nil {
		return
	}

	for _, id := range ids {
		(*theMap)[id] = true
	}
}

func deleteFromMap(ids []string, theMap *map[string]bool) {
	if ids == nil {
		return
	}

	for _, id := range ids {
		delete(*theMap, id)
	}
}

func HandleObjectEditAccessReq(req *protos.ObjectEditAccessReq, hctx wsHelpers.HandlerContext) ([]*protos.ObjectEditAccessResp, error) {
	if err := wsHelpers.CheckStringField(&req.ObjectId, "ObjectId", 1, wsHelpers.IdFieldMaxLength*2 /* Tests have longer ids anyway... */); err != nil {
		return nil, err
	}

	ctx := context.TODO()

	// Determine if we have edit access to the object
	owner, err := wsHelpers.CheckObjectAccess(true, req.ObjectId, req.ObjectType, hctx)
	if err != nil {
		return nil, err
	}

	viewerUsers := map[string]bool{}
	viewerGroups := map[string]bool{}
	editorUsers := map[string]bool{}
	editorGroups := map[string]bool{}

	// Read what's there now
	if owner.Editors != nil {
		readToMap(owner.Editors.UserIds, &editorUsers)
		readToMap(owner.Editors.GroupIds, &editorGroups)
	}
	if owner.Viewers != nil {
		readToMap(owner.Viewers.UserIds, &viewerUsers)
		readToMap(owner.Viewers.GroupIds, &viewerGroups)
	}

	// Add new ones
	if req.AddEditors != nil {
		readToMap(req.AddEditors.UserIds, &editorUsers)
		readToMap(req.AddEditors.GroupIds, &editorGroups)
	}
	if req.AddViewers != nil {
		readToMap(req.AddViewers.UserIds, &viewerUsers)
		readToMap(req.AddViewers.GroupIds, &viewerGroups)
	}

	// Delete ones that need to be deleted
	if req.DeleteEditors != nil {
		deleteFromMap(req.DeleteEditors.UserIds, &editorUsers)
		deleteFromMap(req.DeleteEditors.GroupIds, &editorGroups)
	}

	if req.DeleteViewers != nil {
		deleteFromMap(req.DeleteViewers.UserIds, &viewerUsers)
		deleteFromMap(req.DeleteViewers.GroupIds, &viewerGroups)
	}

	// Put them back into arrays
	viewerUserIds := utils.GetMapKeys(viewerUsers)
	viewerGroupsIds := utils.GetMapKeys(viewerGroups)
	editorUsersIds := utils.GetMapKeys(editorUsers)
	editorGroupsIds := utils.GetMapKeys(editorGroups)

	// Form DB update
	update := bson.D{
		{Key: "viewers", Value: bson.D{
			{Key: "userids", Value: viewerUserIds},
			{Key: "groupids", Value: viewerGroupsIds},
		}},
		{Key: "editors", Value: bson.D{
			{Key: "userids", Value: editorUsersIds},
			{Key: "groupids", Value: editorGroupsIds},
		}},
	}

	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).UpdateByID(ctx, req.ObjectId, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("Ownership UpdateByID result had unexpected counts %+v id: %v, type: %v", result, req.ObjectId, req.ObjectType.String())
	}

	if owner.Editors == nil {
		owner.Editors = &protos.UserGroupList{}
	}

	if owner.Viewers == nil {
		owner.Viewers = &protos.UserGroupList{}
	}

	owner.Editors.UserIds = editorUsersIds
	owner.Editors.GroupIds = editorGroupsIds
	owner.Viewers.UserIds = viewerUserIds
	owner.Viewers.GroupIds = viewerGroupsIds

	return []*protos.ObjectEditAccessResp{&protos.ObjectEditAccessResp{
		Ownership: owner,
	}}, nil
}
