// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package endpoints

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

func GetMemoise(params apiRouter.ApiHandlerGenericParams) error {
	// Get key from query params
	key := params.PathParams["key"]

	// Read from DB, if not there, fail. We do limit key sizes though
	if err := wsHelpers.CheckStringField(&key, "Key", 1, 1024); err != nil {
		return err
	}

	ctx := context.TODO()
	filter := bson.M{"_id": key}
	opts := options.FindOne()
	coll := params.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	result := coll.FindOne(ctx, filter, opts)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return errorwithstatus.MakeNotFoundError(key)
		}
		return result.Err()
	}

	item := &protos.MemoisedItem{}
	err := result.Decode(item)
	if err != nil {
		return err
	}

	now := uint32(params.Svcs.TimeStamper.GetTimeNowSec())

	// Check if this is passed the max age we allow for an item to live in our cache
	if item.LastReadTimeUnixSec < now-uint32(params.Svcs.Config.MaxUnretrievedMemoisationAgeSec) {
		// It's too old, delete & don't return
		params.Svcs.Log.Infof("Retrieved memoised item: %v that hasn't been accessed in %v sec. Deleting.", key, now-item.LastReadTimeUnixSec)

		delResult, err := coll.DeleteOne(ctx, filter, options.Delete())
		if err != nil {
			// Don't error out on this, but do notify
			params.Svcs.Log.Errorf("Failed to delete outdated memoised item: %v. Error: %v", key, err)
		} else {
			if delResult.DeletedCount != 1 {
				params.Svcs.Log.Errorf("Memoised item delete had unexpected counts %+v key: %v", delResult, key)
			}

			return errorwithstatus.MakeNotFoundError(key)
		}
	} else {
		// Update last accessed time here
		if now != item.LastReadTimeUnixSec {
			update := bson.D{{Key: "$set", Value: bson.D{{Key: "lastreadtimeunixsec", Value: now}}}}
			updResult, err := coll.UpdateOne(ctx, filter, update, options.Update())
			if err != nil {
				// Don't error out on this, but do notify
				params.Svcs.Log.Errorf("Failed to update last read time stamp for memoised item: %v. Error: %v", key, err)
			}

			if updResult.ModifiedCount != 1 {
				params.Svcs.Log.Errorf("Memoised item timestamp update had unexpected counts %+v key: %v", updResult, key)
			}

			// Also set it in the item we're replying with
			item.LastReadTimeUnixSec = now
		}
	}

	utils.SendProtoBinary(params.Writer, item)

	return nil
}

func PutMemoise(params apiRouter.ApiHandlerGenericParams) error {
	// Get key from query params
	key := params.PathParams["key"]

	if err := wsHelpers.CheckStringField(&key, "Key", 1, 1024); err != nil {
		return err
	}

	reqData, err := io.ReadAll(params.Request.Body)
	if err != nil {
		return err
	}

	reqItem := &protos.MemoisedItem{}
	err = proto.Unmarshal(reqData, reqItem)
	if err != nil {
		return err
	}

	isClientSavedMap := strings.HasPrefix(key, client.ClientMapKeyPrefix)

	// Ensure key is either empty or the same as the key in the query param
	if len(reqItem.Key) > 0 && key != reqItem.Key {
		return errorwithstatus.MakeBadRequestError(errors.New("Memoisation item key doesn't match query parameter"))
	}

	// Here we overwrite freely, but we do limit key sizes though
	if len(reqItem.Data) <= 0 {
		return errorwithstatus.MakeBadRequestError(errors.New("Missing data field"))
	}

	ctx := context.TODO()
	coll := params.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	opt := options.Update().SetUpsert(true)

	timestamp := uint32(params.Svcs.TimeStamper.GetTimeNowSec())
	item := &protos.MemoisedItem{
		Key:                 reqItem.Key,
		MemoTimeUnixSec:     timestamp,
		Data:                reqItem.Data,
		ScanId:              reqItem.ScanId,
		QuantId:             reqItem.QuantId,
		ExprId:              reqItem.ExprId,
		DataSize:            uint32(len(reqItem.Data)),
		LastReadTimeUnixSec: timestamp, // Right now this is the last time it was accessed. To be updated in future get calls
		MemoWriterUserId:    params.UserInfo.UserID,
	}

	// If we're a client-library side saved map, we don't want this item wiped out by garbage collection!
	if isClientSavedMap {
		item.NoGC = true
	}

	result, err := coll.UpdateByID(ctx, reqItem.Key, bson.D{{Key: "$set", Value: item}}, opt)
	if err != nil {
		return err
	}

	if result.UpsertedCount != 1 {
		params.Svcs.Log.Errorf("MemoiseWriteReq for: %v got unexpected DB write result: %+v", reqItem.Key, result)
	}

	params.Writer.Header().Add("Content-Type", "application/json")

	ts := fmt.Sprintf(`{"timestamp": %v}`, timestamp)
	params.Writer.Write([]byte(ts))

	// If user just saved a client-side created map, notify that it changed as they may be viewing it in a PIXLISE UI instance
	if isClientSavedMap {
		params.Svcs.Notifier.SysNotifyMapChanged(key)
	}

	return nil
}
