package main

import (
	"fmt"

	"github.com/pixlise/core/v2/core/logger"

	"github.com/pixlise/core/v2/core/fileaccess"
	apiNotifications "github.com/pixlise/core/v2/core/notifications"
)

func triggerErrorNotifications(ns apiNotifications.NotificationManager) (string, error) {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/

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

func triggernotifications(fs fileaccess.FileAccess, update bool, updatetype string, notificationStack apiNotifications.NotificationManager, jobLog logger.ILogger) (string, error) {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/
	if notificationStack != nil {
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
	} else {
		fmt.Println("Notification Stack is empty, this is a success notification")
	}
	return "", nil
}
