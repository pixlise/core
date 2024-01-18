package wsHandler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func formUserScreenConfigurationId(user *protos.UserInfo, scanId string) string {
	return user.Id + "-" + scanId
}

func formWidgetId(widget *protos.WidgetLayoutConfiguration, screenConfigId string, layoutIndex int) string {
	positionId := fmt.Sprint(widget.StartRow) + "-" + fmt.Sprint(widget.StartColumn) + "-" + fmt.Sprint(widget.EndRow) + "-" + fmt.Sprint(widget.EndColumn)
	return screenConfigId + "-" + fmt.Sprint(layoutIndex) + "-" + positionId
}

func HandleScreenConfigurationGetReq(req *protos.ScreenConfigurationGetReq, hctx wsHelpers.HandlerContext) (*protos.ScreenConfigurationGetResp, error) {
	configId := ""
	if req.Id != "" {
		configId = req.Id
	} else if req.ScanId != "" {
		configId = formUserScreenConfigurationId(hctx.SessUser.User, req.ScanId)
	}

	screenConfig, owner, err := wsHelpers.GetUserObjectById[protos.ScreenConfiguration](false, configId, protos.ObjectType_OT_SCREEN_CONFIG, dbCollections.ScreenConfigurationName, hctx)
	if err != nil {
		return nil, err
	}

	screenConfig, err = loadWidgetsForScreenConfiguration(screenConfig, hctx)
	if err != nil {
		return nil, err
	}

	screenConfig.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	return &protos.ScreenConfigurationGetResp{
		ScreenConfiguration: screenConfig,
	}, nil
}

func HandleScreenConfigurationListReq(req *protos.ScreenConfigurationListReq, hctx wsHelpers.HandlerContext) (*protos.ScreenConfigurationListResp, error) {
	filter, idToOwner, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_SCREEN_CONFIG, hctx)
	if err != nil {
		return nil, err
	}

	opts := options.Find()

	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.RegionsOfInterestName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	result := []*protos.ScreenConfiguration{}
	err = cursor.All(context.TODO(), &result)
	if err != nil {
		return nil, err
	}

	// Add ownership info
	for _, screenConfig := range result {
		owner, ok := idToOwner[screenConfig.Id]
		if !ok {
			return nil, errors.New("could not find ownership info for screen config")
		}

		screenConfig.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	}

	return &protos.ScreenConfigurationListResp{
		ScreenConfigurations: result,
	}, nil
}

func writeScreenConfiguration(screenConfig *protos.ScreenConfiguration, hctx wsHelpers.HandlerContext, updateExisting bool) (*protos.ScreenConfiguration, error) {
	ctx := context.TODO()
	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	var configuration *protos.ScreenConfiguration
	var owner *protos.OwnershipItem

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var err error

		if updateExisting {
			configuration, owner, err = wsHelpers.GetUserObjectById[protos.ScreenConfiguration](true, screenConfig.Id, protos.ObjectType_OT_SCREEN_CONFIG, dbCollections.ScreenConfigurationName, hctx)
			if err != nil {
				return nil, err
			}

			updatedConfig := bson.D{}

			if screenConfig.Layouts != nil && len(screenConfig.Layouts) > 0 {
				updatedConfig = append(updatedConfig, bson.E{Key: "layouts", Value: screenConfig.Layouts})
				configuration.Layouts = screenConfig.Layouts

				// Add an ID to any widgets that don't have one
				for i, layout := range screenConfig.Layouts {
					for _, widget := range layout.Widgets {
						if widget.Id == "" {
							widget.Id = formWidgetId(widget, screenConfig.Id, i)
						}
					}
				}
			}

			// These fields can be empty, so we don't need to check for them
			updatedConfig = append(updatedConfig, bson.E{Key: "name", Value: screenConfig.Name})
			updatedConfig = append(updatedConfig, bson.E{Key: "tags", Value: screenConfig.Tags})
			updatedConfig = append(updatedConfig, bson.E{Key: "description", Value: screenConfig.Description})
			updatedConfig = append(updatedConfig, bson.E{Key: "scanconfigurations", Value: screenConfig.ScanConfigurations})

			configuration.Name = screenConfig.Name
			configuration.Tags = screenConfig.Tags
			configuration.Description = screenConfig.Description
			configuration.ScanConfigurations = screenConfig.ScanConfigurations

			_, err = hctx.Svcs.MongoDB.Collection(dbCollections.ScreenConfigurationName).UpdateByID(sessCtx, screenConfig.Id, bson.D{{
				Key:   "$set",
				Value: updatedConfig,
			}})
		} else {
			// In order for a screen config to be valid, we must at least have one layout
			if screenConfig.Layouts == nil || len(screenConfig.Layouts) <= 0 {
				return nil, errors.New("screen configuration must have at least one layout")
			}

			// Add an ID to any widgets that don't have one
			for i, layout := range screenConfig.Layouts {
				for _, widget := range layout.Widgets {
					if widget.Id == "" {
						widget.Id = formWidgetId(widget, screenConfig.Id, i)
					}
				}
			}

			// We need to create an ownership item along with it
			owner, err = wsHelpers.MakeOwnerForWrite(screenConfig.Id, protos.ObjectType_OT_SCREEN_CONFIG, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())
			if err != nil {
				return nil, err
			}

			screenConfig.ModifiedUnixSec = owner.CreatedUnixSec
			configuration = screenConfig

			_, err = hctx.Svcs.MongoDB.Collection(dbCollections.ScreenConfigurationName).InsertOne(sessCtx, screenConfig)
			if err != nil {
				return nil, err
			}

			_, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, owner)
		}

		if err != nil {
			return nil, err
		}

		return nil, err
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return nil, err
	}

	configuration.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	return configuration, nil
}

func checkIfScreenConfigurationExists(id string, hctx wsHelpers.HandlerContext, canEdit bool) (bool, error) {
	_, _, err := wsHelpers.GetUserObjectById[protos.ScreenConfiguration](true, id, protos.ObjectType_OT_SCREEN_CONFIG, dbCollections.ScreenConfigurationName, hctx)
	if err != nil {
		switch e := err.(type) {
		case errorwithstatus.Error:
			if e.Status() == http.StatusNotFound {
				// This is a not found error!
				return false, nil
			}
		}

		// Something else went wrong
		return false, err
	}

	return true, nil
}

func loadWidgetsForScreenConfiguration(screenConfig *protos.ScreenConfiguration, hctx wsHelpers.HandlerContext) (*protos.ScreenConfiguration, error) {
	if screenConfig == nil || screenConfig.Layouts == nil || len(screenConfig.Layouts) == 0 {
		return screenConfig, nil
	}

	ctx := context.TODO()
	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var err error

		for layoutIndex, layout := range screenConfig.Layouts {
			for widgetIndex, widget := range layout.Widgets {
				if widget.Id != "" {
					result := hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).FindOne(sessCtx, bson.M{
						"_id": widget.Id,
					})

					widgetData := &protos.WidgetData{}
					if result.Err() != nil {
						if result.Err() == mongo.ErrNoDocuments {
							// Widget not found, insert a new one with this ID
							_, err := hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).InsertOne(sessCtx, &protos.WidgetData{
								Id: widget.Id,
							})
							if err != nil {
								return nil, err
							}

							widgetData.Id = widget.Id
						} else {
							return nil, result.Err()
						}
					} else {
						err = result.Decode(&widgetData)
						if err != nil {
							return nil, err
						}
					}

					screenConfig.Layouts[layoutIndex].Widgets[widgetIndex].Data = widgetData
				}
			}
		}

		return nil, err
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return screenConfig, err
	}

	return screenConfig, nil
}

func HandleScreenConfigurationWriteReq(req *protos.ScreenConfigurationWriteReq, hctx wsHelpers.HandlerContext) (*protos.ScreenConfigurationWriteResp, error) {
	if req.ScreenConfiguration == nil {
		return nil, errors.New("screen configuration must be specified")
	}

	if req.ScreenConfiguration.Layouts == nil || len(req.ScreenConfiguration.Layouts) == 0 {
		return nil, errors.New("screen configuration must have at least one layout")
	}

	screenConfig := req.ScreenConfiguration

	updateExisting := req.ScreenConfiguration.Id != ""

	if req.ScreenConfiguration.Id == "" {
		if req.ScanId != "" {
			screenConfig.Id = formUserScreenConfigurationId(hctx.SessUser.User, req.ScanId)
			exists, err := checkIfScreenConfigurationExists(screenConfig.Id, hctx, true)
			if err != nil {
				return nil, err
			}

			// If it exists, we'll update it, otherwise we'll create a new one
			updateExisting = exists
		} else {
			// Generate a new id
			screenConfig.Id = hctx.Svcs.IDGen.GenObjectID()
		}
	}

	screenConfig, err := writeScreenConfiguration(screenConfig, hctx, updateExisting)
	if err != nil {
		return nil, err
	}

	screenConfig, err = loadWidgetsForScreenConfiguration(screenConfig, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.ScreenConfigurationWriteResp{
		ScreenConfiguration: screenConfig,
	}, nil
}

func HandleScreenConfigurationDeleteReq(req *protos.ScreenConfigurationDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ScreenConfigurationDeleteResp, error) {
	if req.Id == "" {
		return nil, errors.New("screen configuration id must be specified")
	}

	// Check if exists and if user can delete
	screenConfig, _, err := wsHelpers.GetUserObjectById[protos.ScreenConfiguration](true, req.Id, protos.ObjectType_OT_SCREEN_CONFIG, dbCollections.ScreenConfigurationName, hctx)
	if err != nil {
		return nil, err
	}

	// Run through all the widgets and delete them, then delete screen config and ownership item
	ctx := context.TODO()
	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var err error

		for _, layout := range screenConfig.Layouts {
			for _, widget := range layout.Widgets {
				if widget.Id != "" {
					_, err = hctx.Svcs.MongoDB.Collection(dbCollections.WidgetDataName).DeleteOne(sessCtx, bson.M{
						"_id": widget.Id,
					})
					if err != nil {
						return nil, err
					}
				}
			}
		}

		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.ScreenConfigurationName).DeleteOne(sessCtx, bson.M{
			"_id": req.Id,
		})
		if err != nil {
			return nil, err
		}

		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).DeleteOne(sessCtx, bson.M{
			"_id": req.Id,
		})
		if err != nil {
			return nil, err
		}

		return nil, err
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return nil, err
	}

	return &protos.ScreenConfigurationDeleteResp{
		Id: req.Id,
	}, nil
}
