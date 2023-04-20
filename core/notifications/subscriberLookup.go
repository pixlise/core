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

package notifications

import (
	"context"
	"errors"
	"strings"

	"github.com/pixlise/core/v3/core/pixlUser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func readUsers(userCollection *mongo.Collection, filter interface{}) ([]pixlUser.UserStruct, error) {
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	users := []pixlUser.UserStruct{}

	for cursor.Next(context.Background()) {
		l := pixlUser.UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		users = append(users, l)
	}

	return users, nil
}

func (n *NotificationStack) getAllUsers() ([]pixlUser.UserStruct, error) {
	if n.userCollection == nil {
		return nil, errors.New("getAllUsers: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching All Subscribers Mongo Object")

	filter := bson.D{}
	users, err := readUsers(n.userCollection, filter)

	//n.Logger.Debugf("Fetched All Subscribers Mongo Object")
	return users, err
}

func (n *NotificationStack) getSubscribersByTopicID(useroverride []string, searchtopic string) ([]pixlUser.UserStruct, error) {
	if n.userCollection == nil {
		return nil, errors.New("getSubscribersByTopicID: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching Subscriber Mongo Object for topic: %v", searchtopic)

	var filter bson.M
	if useroverride != nil && len(useroverride) > 0 {
		var v []string
		for _, f := range useroverride {
			s := strings.TrimPrefix(f, "auth0|")
			v = append(v, s)
		}
		filter = bson.M{
			"$and": []bson.D{
				{
					{"userid", bson.D{{"$in", v}}},
				},
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	} else {
		filter = bson.M{
			"$and": []bson.D{
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	}

	users, err := readUsers(n.userCollection, filter)

	//n.Logger.Debugf("Fetched Subscriber Mongo Object for topic: %v", searchtopic)

	return users, err
}

func (n *NotificationStack) getSubscribersByEmailTopicID(useroverride []string, searchtopic string) ([]pixlUser.UserStruct, error) {
	if n.userCollection == nil {
		return nil, errors.New("getSubscribersByEmailTopicID: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching Subscriber Mongo Object for topic: %v", searchtopic)

	var filter bson.M
	if useroverride != nil && len(useroverride) > 0 {
		var v []string
		for _, f := range useroverride {
			s := strings.TrimPrefix(f, "auth0|")
			v = append(v, s)
		}
		filter = bson.M{
			"$and": []bson.D{
				{
					{"userconfig.email", bson.D{{"$in", useroverride}}},
				},
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	} else {
		filter = bson.M{
			"$and": []bson.D{
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	}

	users, err := readUsers(n.userCollection, filter)

	//n.Logger.Debugf("Fetched Subscriber Mongo Object for topic: %v", searchtopic)

	return users, err
}

func (n *NotificationStack) getSubscribersByTopic(searchtopic string) ([]pixlUser.UserStruct, error) {
	if n.userCollection == nil {
		return nil, errors.New("getSubscribersByTopic: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching Subscribers Mongo Object for topic: %v", searchtopic)

	var filter bson.M

	filter = bson.M{
		"$and": []bson.D{
			{
				{
					Key:   "notifications.topics.name",
					Value: searchtopic,
				},
			},
		},
	}

	users, err := readUsers(n.userCollection, filter)

	//n.Logger.Debugf("Fetched Subscribers Mongo Object for topic: %v", searchtopic)

	return users, err
}
