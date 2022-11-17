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

package quantModel

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v2/core/notifications"

	"github.com/pixlise/core/v2/core/pixlUser"
)

func startQuantNotification(params PiquantParams, notificationStack notifications.NotificationManager, creator pixlUser.UserInfo) error {
	note := notifications.UINotificationItem{
		Topic: "Quantification Processing Start",
		//                                    let msg = 'Started Quantification: "'+result.quantName+'" (id: '+jobId+'). Click on Quant Logs tab to follow progress.';
		Message:   fmt.Sprintf("Started Quantification: %v (id: %v). Click on Quant Tracker tab to follow progress.", params.QuantName, params.JobID),
		Timestamp: time.Time{},
		UserID:    creator.UserID,
	}
	return notificationStack.SendUINotification(note)
}

func endQuantNotification(quantname string, notificationStack notifications.NotificationManager, creator pixlUser.UserInfo) error {
	template := make(map[string]interface{})
	template["quantname"] = quantname
	template["subject"] = fmt.Sprintf("Your quantification job, %v, has completed.", quantname)
	users := []string{"auth0|" + creator.UserID}

	fmt.Println("Wait over, dispatching notification for: " + creator.UserID)
	err := notificationStack.SendAll("user-quant-complete", template, users, false)
	note := notifications.UINotificationItem{
		Topic:     "Quantification Processing Complete",
		Message:   fmt.Sprintf("Quantification %v Processing Complete", quantname),
		Timestamp: time.Time{},
		UserID:    creator.UserID,
	}
	err = notificationStack.SendUINotification(note)
	return err
}

func quantFailedNotification(quantname string, notificationStack notifications.NotificationManager, userid string) error {
	template := make(map[string]interface{})
	template["quantname"] = quantname
	template["subject"] = fmt.Sprintf("Quantification job, %v, has FAILED.", quantname)
	users := []string{"auth0|" + userid}

	fmt.Println("Wait over, dispatching notification for: " + userid)
	err := notificationStack.SendAll("quant-processing-failed", template, users, true)
	note := notifications.UINotificationItem{
		Topic:     "Quantification Processing Complete",
		Message:   fmt.Sprintf("Quantification %v Processing Failed", quantname),
		Timestamp: time.Time{},
		UserID:    userid,
	}
	err = notificationStack.SendUINotification(note)

	return err
}
