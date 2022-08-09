package main

import (
	"fmt"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"

	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	apiNotifications "gitlab.com/pixlise/pixlise-go-api/core/notifications"
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
