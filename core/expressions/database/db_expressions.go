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

package expressionDB

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/expressions/expressions"
	"github.com/pixlise/core/v3/core/pixlUser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (e *ExpressionDB) ListExpressions(userID string, includeShared bool, retrieveUpdatedUserInfo bool) (expressions.DataExpressionLookup, error) {
	result := expressions.DataExpressionLookup{}
	if e.Expressions == nil {
		return result, errors.New("ListExpressions: Mongo not connected")
	}

	// List all which have this user ID
	filter := bson.D{primitive.E{Key: "origin.creator.userid", Value: userID}}
	if includeShared {
		filter = bson.D{
			{"$or",
				bson.A{
					bson.D{{"origin.creator.userid", userID}},
					bson.D{{"origin.shared", true}},
				},
			},
		}
	}
	opts := options.Find().SetProjection(bson.D{{"sourcecode", 0}})
	cursor, err := e.Expressions.Find(context.TODO(), filter, opts)

	if err != nil {
		return result, err
	}

	allExpressions := []expressions.DataExpression{}
	err = cursor.All(context.TODO(), &allExpressions)
	if err != nil {
		return result, err
	}

	// Return them as a map while also fixing up user info
	for _, exprItem := range allExpressions {
		if retrieveUpdatedUserInfo {
			// Get latest user details from Mongo
			updatedCreator, creatorErr := e.Svcs.Users.GetCurrentCreatorDetails(exprItem.Origin.Creator.UserID)
			if creatorErr != nil {
				e.Svcs.Log.Infof("Failed to lookup user details for ID: %v (Name: %v), creator name in expression: %v. Error: %v", exprItem.Origin.Creator.UserID, exprItem.Origin.Creator.Name, exprItem.ID, creatorErr)
			} else {
				exprItem.Origin.Creator = updatedCreator
			}
		}

		// Save in map
		result[exprItem.ID] = exprItem
	}

	return result, nil
}

func (e *ExpressionDB) GetExpression(expressionID string, retrieveUpdatedUserInfo bool) (expressions.DataExpression, error) {
	result := expressions.DataExpression{}

	exprResult := e.Expressions.FindOne(context.TODO(), bson.M{"_id": expressionID})

	if exprResult.Err() != nil {
		return result, exprResult.Err()
	}

	// Read the expression item
	err := exprResult.Decode(&result)
	if err != nil {
		return result, err
	}

	if retrieveUpdatedUserInfo {
		// Get latest user details from Mongo
		updatedCreator, creatorErr := e.Svcs.Users.GetCurrentCreatorDetails(result.Origin.Creator.UserID)
		if creatorErr != nil {
			e.Svcs.Log.Infof("Failed to lookup user details for ID: %v (Name: %v), creator name in expression: %v. Error: %v", result.Origin.Creator.UserID, result.Origin.Creator.Name, result.ID, creatorErr)
		} else {
			result.Origin.Creator = updatedCreator
		}
	}

	return result, err
}

func (e *ExpressionDB) CreateExpression(input expressions.DataExpressionInput, creator pixlUser.UserInfo, createShared bool) (expressions.DataExpression, error) {
	nowUnix := e.Svcs.TimeStamper.GetTimeNowSec()
	exprID := e.Svcs.IDGen.GenObjectID()

	expr := expressions.DataExpression{
		ID:               exprID,
		Name:             input.Name,
		SourceCode:       input.SourceCode,
		SourceLanguage:   input.SourceLanguage,
		Comments:         input.Comments,
		Tags:             input.Tags,
		ModuleReferences: input.ModuleReferences,
		Origin: pixlUser.APIObjectItem{
			Shared:              createShared,
			Creator:             creator,
			CreatedUnixTimeSec:  nowUnix,
			ModifiedUnixTimeSec: nowUnix,
		},
		// RecentExecStats is blank at this point!
	}

	// Write the expression itself
	insertResult, err := e.Expressions.InsertOne(context.TODO(), expr)
	if err != nil {
		return expr, err
	}
	if insertResult.InsertedID != exprID {
		e.Svcs.Log.Errorf("CreateExpression: Expected Mongo insert to return ID %v, got %v", exprID, insertResult.InsertedID)
	}
	return expr, nil
}

// Replaces the existing expression with the new one
// This assumes a GetExpression was required already to validate user permissions to this expression, etc
// therefore the prevUnixTime should be available. This way we can preserve the creation time but set a new
// modified time now. Also now requires the existing expression shared and source code field!
func (e *ExpressionDB) UpdateExpression(
	expressionID string,
	input expressions.DataExpressionInput,
	creator pixlUser.UserInfo,
	createdUnixTimeSec int64,
	isShared bool,
	existingSourceCode string,
) (expressions.DataExpression, error) {
	filter := bson.D{{"_id", expressionID}}

	// NOTE: originally wanted to just edit the new fields, like so:
	/*
		update := bson.D{
			{"$set", bson.D{
				{"name", input.Name},
				{"sourceCode", input.SourceCode},
				{"sourceLanguage", input.SourceLanguage},
				{"comments", input.Comments},
				{"tags", input.Tags},
			}},
		}
		result, err := e.Expressions.UpdateOne(context.TODO(), filter, update)
	*/
	// But the above seemed risky because BSON field names may change as the struct changes, but this code wouldn't!
	// So instead opting to replace with a new record as this will obey struct field naming
	// Only complication is we've since introduced the idea of sending in a blank string for source code meaning preserve the old code
	// so we have to read the existing expression here in that case:
	sourceCode := input.SourceCode
	if len(sourceCode) <= 0 {
		// Use existing source field...
		sourceCode = existingSourceCode
	}

	if len(sourceCode) <= 0 {
		return expressions.DataExpression{}, fmt.Errorf("Expression source code field cannot be blank, when updating expression: %v", expressionID)
	}

	nowUnix := e.Svcs.TimeStamper.GetTimeNowSec()
	replacement := expressions.DataExpression{
		ID:               expressionID,
		Name:             input.Name,
		SourceCode:       sourceCode,
		SourceLanguage:   input.SourceLanguage,
		Comments:         input.Comments,
		Tags:             input.Tags,
		ModuleReferences: input.ModuleReferences,
		Origin: pixlUser.APIObjectItem{
			Shared:              isShared,
			Creator:             creator,
			CreatedUnixTimeSec:  createdUnixTimeSec,
			ModifiedUnixTimeSec: nowUnix,
		},
		// Expression was edited, so any previous RecentExecStats are no longer valid, so blank!
	}

	updResult, err := e.Expressions.ReplaceOne(context.TODO(), filter, replacement)
	if err != nil {
		return replacement, err
	}
	if updResult.MatchedCount != 1 || updResult.ModifiedCount != 1 {
		e.Svcs.Log.Errorf("UpdateExpression for %v: Expected Mongo ReplaceOne to modify 1 item, got match: %v, modified: %v", expressionID, updResult.MatchedCount, updResult.ModifiedCount)
	}
	return replacement, nil
}

func (e *ExpressionDB) StoreExpressionRecentRunStats(expressionID string, stats expressions.DataExpressionExecStats) error {
	filter := bson.D{{"_id", expressionID}}

	update := bson.D{
		{"$set", bson.D{
			{"recentExecStats", stats},
		}},
	}
	updResult, err := e.Expressions.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}
	// Make sure it worked
	if updResult.MatchedCount != 1 || updResult.ModifiedCount != 1 {
		return api.MakeNotFoundError(expressionID)
		//err = fmt.Errorf("StoreExpressionRecentRunStats for %v: Expected Mongo ReplaceOne to modify 1 item, got match: %v, modified: %v", expressionID, updResult.MatchedCount, updResult.ModifiedCount)
		//return err
	}
	return nil
}

func (e *ExpressionDB) DeleteExpression(expressionID string) error {
	delResult, err := e.Expressions.DeleteOne(context.TODO(), bson.M{"_id": expressionID})

	if err != nil {
		return err
	}

	// Should really get 1 deletion... Or 0 is still valid...
	if delResult.DeletedCount > 1 {
		e.Svcs.Log.Errorf("DeleteExpression %v: expected to delete 1 item, instead deleted %v", expressionID, delResult.DeletedCount)
	}

	return nil
}
