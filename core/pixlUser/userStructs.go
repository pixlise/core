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

package pixlUser

import "time"

// UINotificationItem - A single UI Notification
type UINotificationItem struct {
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userid"`
}

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
type UserDetails struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Cell           string `json:"cell"`
	DataCollection string `json:"data_collection"`
}

// UserStruct - Structure for user configuration
type UserStruct struct {
	Userid        string        `json:"userid"`
	Notifications Notifications `json:"notifications"`
	Config        UserDetails   `json:"userconfig"`
}
