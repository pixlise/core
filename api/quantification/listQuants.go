package quantification

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ListUserQuants(searchParams *protos.SearchParams, hctx wsHelpers.HandlerContext) ([]*protos.QuantificationSummary, map[string]*protos.OwnershipItem, error) {
	filter, idToOwner, err := wsHelpers.MakeFilter(searchParams, false, protos.ObjectType_OT_QUANTIFICATION, hctx)
	if err != nil {
		return nil, idToOwner, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.QuantificationsName)

	opts := options.Find()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, idToOwner, err
	}

	items := []*protos.QuantificationSummary{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, idToOwner, err
	}

	return items, idToOwner, nil
}
