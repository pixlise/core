package quantification

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ListUserQuants(searchParams *protos.SearchParams, svcs *services.APIServices, sessUser sessionuser.SessionUser) ([]*protos.QuantificationSummary, map[string]*protos.OwnershipItem, error) {
	filter, idToOwner, err := wsHelpers.MakeFilterForUser(searchParams, false, protos.ObjectType_OT_QUANTIFICATION, svcs, sessUser)
	if err != nil {
		return nil, idToOwner, err
	}

	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.QuantificationsName)

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
