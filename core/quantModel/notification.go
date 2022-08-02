// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package quantModel

import (
	"fmt"
	"github.com/pixlise/core/core/notifications"
	"time"

	"github.com/pixlise/core/core/pixlUser"
)

func startQuantNotification(params PiquantParams, notificationStack notifications.NotificationManager, creator pixlUser.UserInfo) error {
	note := notifications.UINotificationObj{
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
	note := notifications.UINotificationObj{
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
	note := notifications.UINotificationObj{
		Topic:     "Quantification Processing Complete",
		Message:   fmt.Sprintf("Quantification %v Processing Failed", quantname),
		Timestamp: time.Time{},
		UserID:    userid,
	}
	err = notificationStack.SendUINotification(note)

	return err
}
