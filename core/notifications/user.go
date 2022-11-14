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

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Method - Notification Methods
type Method struct {
	UI    bool `json:"ui"`
	Sms   bool `json:"sms"`
	Email bool `json:"email"`
}

// NotificationConfig - Config specifically for notifications
type NotificationConfig struct {
	Method `json:"method"`
}

// Topics - Notification Topics'
type Topics struct {
	Name   string             `json:"name"`
	Config NotificationConfig `json:"config"`
}

// Notifications - Object for notification settings
type Notifications struct {
	Topics          []Topics             `json:"topics"`
	Hints           []string             `json:"hints"`
	UINotifications []UINotificationItem `json:"uinotifications"`
}

// Config - config options for user
type Config struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Cell           string `json:"cell"`
	DataCollection string `json:"data_collection"`
}

// UserStruct - Structure for user configuration
type UserStruct struct {
	Userid        string        `json:"userid"`
	Notifications Notifications `json:"notifications"`
	Config        Config        `json:"userconfig"`
}

func initUser(name string, email string, userid string) UserStruct {
	userConfig := Config{
		DataCollection: "unknown",
		Name:           name,
		Email:          email,
		Cell:           "", // TODO: delete this? It went unused
	}
	userNotifications := Notifications{
		Topics:          []Topics{},
		Hints:           []string{},
		UINotifications: []UINotificationItem{},
	}

	user := UserStruct{
		Userid:        userid,
		Notifications: userNotifications,
		Config:        userConfig,
	}

	return user
}

func (n *NotificationStack) createUser(userid string, name string, email string) (UserStruct, error) {
	us := initUser(name, email, userid)

	if n.notificationCollection == nil {
		return us, errors.New("createUser: Mongo not connected")
	}

	//n.Logger.Debugf("Creating Mongo Object for user: %v", user.Userid)

	_, err := n.userCollection.InsertOne(context.TODO(), us)

	//n.Logger.Debugf("Created Mongo Object for user: %v", user.Userid)

	return us, err
}

func readUsers(userCollection *mongo.Collection, filter interface{}) ([]UserStruct, error) {
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var users []UserStruct

	for cursor.Next(context.Background()) {
		l := UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		users = append(users, l)
	}

	return users, nil
}

func (n *NotificationStack) getAllUsers() ([]UserStruct, error) {
	if n.notificationCollection == nil {
		return nil, errors.New("getAllUsers: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching All Subscribers Mongo Object")

	filter := bson.D{}
	users, err := readUsers(n.userCollection, filter)

	//n.Logger.Debugf("Fetched All Subscribers Mongo Object")
	return users, err
}

func (n *NotificationStack) getSubscribersByTopicID(useroverride []string, searchtopic string) ([]UserStruct, error) {
	if n.notificationCollection == nil {
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

func (n *NotificationStack) getSubscribersByEmailTopicID(useroverride []string, searchtopic string) ([]UserStruct, error) {
	if n.notificationCollection == nil {
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

func (n *NotificationStack) getSubscribersByTopic(searchtopic string) ([]UserStruct, error) {
	if n.notificationCollection == nil {
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

func (n *NotificationStack) GetUser(userid string) (UserStruct, error) {
	if n.notificationCollection == nil {
		return UserStruct{}, errors.New("GetUser: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching Mongo Object for user: %v", userid)

	filter := bson.D{{"userid", userid}}

	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.FindOne().SetSort(sort) //.SetProjection(projection)
	cursor := n.userCollection.FindOne(context.TODO(), filter, opts)

	var user UserStruct

	//n.Logger.Debugf("Decoding Mongo Object for user: %v", userid)
	err := cursor.Decode(&user)
	//n.Logger.Debugf("Fetched Mongo Object for user: %v", userid)

	return user, err
}

func (n *NotificationStack) GetUserEnsureExists(userid string, name string, email string) (UserStruct, error) {
	// Try to read it, if it doesn't exist, create it
	user, err := n.GetUser(userid)
	if err == nil {
		return user, nil
	}

	return n.createUser(userid, name, email)
}

func (n *NotificationStack) WriteUser(user UserStruct) error {
	if n.notificationCollection == nil {
		return errors.New("WriteUser: Mongo not connected")
	}

	//n.Logger.Debugf("Updating Mongo Object for user: %v", userid)

	filter := bson.D{{"userid", user.Userid}}
	update := bson.D{{"$set", user}}
	opts := options.Update().SetUpsert(true)

	_, err := n.userCollection.UpdateOne(context.TODO(), filter, update, opts)
	//n.Logger.Debugf("Updated Mongo Object for user: %v", userid)

	return err
}
