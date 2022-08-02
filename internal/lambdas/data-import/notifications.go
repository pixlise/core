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

package main

import (
	"fmt"
	"github.com/pixlise/core/core/logger"

	"github.com/pixlise/core/core/fileaccess"
	apiNotifications "github.com/pixlise/core/core/notifications"
)

func triggerErrorNotifications(ns apiNotifications.NotificationManager) (string, error) {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/

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

	return "", nil
}

func triggernotifications(fs fileaccess.FileAccess, update bool, updatetype string, notificationStack apiNotifications.NotificationManager, jobLog logger.ILogger) (string, error) {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/
	var err error

	template := make(map[string]interface{})
	dsname, err := computeName()
	if err != nil {
		jobLog.Infof(err.Error())
		return "", err
	}
	template["datasourcename"] = dsname

	if update {
		loads, b := lookupLoadtime(dsname, fs)
		if b {
			jobLog.Infof("Last sent a notification within an hour not resending\n")
		} else {
			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", template["datasourcename"])
			fmt.Println("Wait over, dispatching notification for updated datasource")
			err = notificationStack.SendAllDataSource(fmt.Sprintf("dataset-%v-updated", updatetype), template, nil, true, "dataset-updated")
			if err != nil {
				jobLog.Infof(err.Error())
				return "", err
			}

			saveLoadtime(dsname, loads, fs)
		}
	} else {
		loads, b := lookupLoadtime(dsname, fs)
		if b {
			jobLog.Infof("Last sent a notification within an hour not resending\n")
		} else {
			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", "")

			fmt.Println("Wait over, dispatching notification for new datasource")
			err = notificationStack.SendAllDataSource("new-dataset-available", template, nil, true, "")
			if err != nil {
				jobLog.Infof(err.Error())
				return "", err
			}
			saveLoadtime(dsname, loads, fs)
		}
	}
	return "", nil
}
