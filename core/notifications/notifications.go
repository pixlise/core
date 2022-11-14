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
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UINotificationItem - A single UI Notification
type UINotificationItem struct {
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userid"`
}

func readUINotifications(notificationCollection *mongo.Collection, filter interface{}) ([]UINotificationItem, error) {
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := notificationCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UINotificationItem

	for cursor.Next(context.Background()) {
		l := UINotificationItem{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

// SendUINotification - Dispatch notification to the stack
func (n *NotificationStack) SendUINotification(newNotification UINotificationItem) error {
	if n.notificationCollection == nil {
		return errors.New("SendUINotification: Mongo not connected")
	}

	//n.Logger.Debugf("Inserting UI Notification Mongo Object for user: %v", newNotification.UserID)

	newNotification.Timestamp = time.Unix(n.timestamper.GetTimeNowSec(), 0)
	_, err := n.notificationCollection.InsertOne(context.TODO(), newNotification)
	if err != nil {
		return err
	}

	//n.Logger.Debugf("Inserted UI Notification Mongo Object for user: %v", newNotification.UserID)

	return nil
}

// GetUINotifications - Return Notifications for user and remove from stack
func (n *NotificationStack) GetUINotifications(userid string) ([]UINotificationItem, error) {
	if n.notificationCollection == nil {
		return nil, errors.New("GetUINotifications: Mongo not connected")
	}

	//n.Logger.Debugf("GetUINotifications for user: %v", userid)

	filter := bson.D{{"userid", userid}}
	notifications, err := readUINotifications(n.notificationCollection, filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to read notifications: %v", err)
	}

	//n.Logger.Debugf("Got UINotifications for user: %v", userid)

	// Once we've got them, delete them!
	if err == nil && len(notifications) > 0 {
		_, err = n.notificationCollection.DeleteMany(context.TODO(), filter)

		if err != nil {
			return nil, fmt.Errorf("Failed to delete notifications after retrieval: %v", err)
		}
	}

	return notifications, nil
}
