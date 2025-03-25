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

	"github.com/pixlise/core/v4/api/dbCollections"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
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

	filter := bson.M{"_id": key}
	opts := options.FindOne()
	result := params.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), filter, opts)
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

	utils.SendProtoBinary(params.Writer, item)
	return nil
}

func PutMemoise(params apiRouter.ApiHandlerGenericParams) error {
	reqData, err := io.ReadAll(params.Request.Body)
	if err != nil {
		return err
	}

	reqItem := &protos.MemoisedItem{}
	err = proto.Unmarshal(reqData, req)
	if err != nil {
		return err
	}

	// Here we overwrite freely, but we do limit key sizes though
	if err := wsHelpers.CheckStringField(&reqItem.Key, "Key", 1, 1024); err != nil {
		return err
	}
	if len(reqItem.Data) <= 0 {
		return errorwithstatus.MakeBadRequestError(errors.New("Missing data field"))
	}

	ctx := context.TODO()
	coll := params.Svcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
	opt := options.Update().SetUpsert(true)

	timestamp := uint32(params.Svcs.TimeStamper.GetTimeNowSec())
	item := &protos.MemoisedItem{
		Key:             reqItem.Key,
		MemoTimeUnixSec: timestamp,
		Data:            reqItem.Data,
		ScanId:          reqItem.ScanId,
		QuantId:         reqItem.QuantId,
		ExprId:          reqItem.ExprId,
	}

	result, err := coll.UpdateByID(ctx, reqItem.Key, bson.D{{Key: "$set", Value: item}}, opt)
	if err != nil {
		return err
	}

	if result.UpsertedCount != 1 {
		params.Svcs.Log.Errorf("MemoiseWriteReq for: %v got unexpected DB write result: %+v", reqItem.Key, result)
	}

	params.Writer.Header().Add("Content-Type", "application/json")

	ts := fmt.Sprintf("%v", timestamp)
	params.Writer.Write([]byte(ts))

	return nil
}
