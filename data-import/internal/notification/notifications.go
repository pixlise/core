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

package importerNotification

import (
	"errors"
	"fmt"
	"time"

	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/data-import/internal/importtime"

	"github.com/pixlise/core/v2/core/fileaccess"
	apiNotifications "github.com/pixlise/core/v2/core/notifications"
)

/*
func triggerErrorNotifications(ns apiNotifications.NotificationManager) (string, error) {
	/ * var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}* /

	if ns != nil {
		fmt.Println("Wait over, dispatching notification for new datasource")
		template := make(map[string]interface{})
		var err error
		template["datasourcename"], err = computeName()
		if err != nil {
			return "", err
		}
		template["subject"] = fmt.Sprintf("Error Processing Datasource %v", template["datasourcename"])
		err = ns.SendAllDataSource("updated-dataset-available", template, nil, true, "new-datasource-available")
		if err != nil {
			return "", err
		}
	} else {
		fmt.Println("Notification Stack is empty, this is an error notification")
	}
	return "", nil
}
*/

func TriggerNotifications(
	configBucket string,
	datasetName string,
	fs fileaccess.FileAccess,
	update bool,
	updatetype string,
	notificationStack apiNotifications.NotificationManager,
	jobLog logger.ILogger) error {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/
	if notificationStack == nil {
		return errors.New("Notification Stack is empty, this is a success notification")
	}
	var err error

	template := make(map[string]interface{})
	template["datasourcename"] = datasetName

	lastImportUnixSec, err := importtime.GetDatasetImportUnixTimeSec(fs, configBucket, datasetName)

	// Print an error if we got one, but this can always continue...
	if err != nil {
		jobLog.Errorf("%v", err)
	}

	lastImportTime := time.Unix(int64(lastImportUnixSec), 0)
	if time.Since(lastImportTime).Minutes() < 60 {
		jobLog.Infof("Skipping notification send - one was sent recently")
	} else {
		if update {
			jobLog.Infof("Dispatching notification for updated datasource")

			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", template["datasourcename"])
			err = notificationStack.SendAllDataSource(fmt.Sprintf("dataset-%v-updated", updatetype), template, nil, true, "dataset-updated")
		} else {
			jobLog.Infof("Dispatching notification for new datasource")

			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", "")
			err = notificationStack.SendAllDataSource("new-dataset-available", template, nil, true, "")
		}
	}

	tsSaveErr := importtime.SaveDatasetImportUnixTimeSec(fs, jobLog, configBucket, datasetName, int(time.Now().Unix()))

	if tsSaveErr != nil {
		jobLog.Errorf(tsSaveErr.Error())
	}

	// Also write out
	if err != nil {
		jobLog.Errorf(err.Error())
	}
	return err
}
