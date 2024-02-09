package notificationSender

import (
	"fmt"
	"path"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/ws"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationSender struct {
	instanceId  string
	db          *mongo.Database
	timestamper timestamper.ITimeStamper // So we can mock time.Now()
	log         logger.ILogger
	envRootURL  string
	ws          *ws.WSHandler
	melody      *melody.Melody
}

func MakeNotificationSender(instanceId string, db *mongo.Database, timestamper timestamper.ITimeStamper, log logger.ILogger, envRootURL string, ws *ws.WSHandler, melody *melody.Melody) *NotificationSender {
	return &NotificationSender{
		instanceId:  instanceId,
		db:          db,
		timestamper: timestamper,
		log:         log,
		ws:          ws,
		melody:      melody,
	}
}

func (n *NotificationSender) NotifyNewScan(scanName string, scanId string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          fmt.Sprintf("New scan imported: %v", scanName),
			Contents:         fmt.Sprintf("A new scan named %v was just imported. Scan ID is: %v", scanName, scanId),
			From:             "Data Importer",
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       path.Join(n.envRootURL, "?q="+scanId),
			Meta:             map[string]string{},
		},
	}

	n.sendNotificationToObjectUsers(notifMsg, scanId)
}

func (n *NotificationSender) NotifyUpdatedScan(scanName string, scanId string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          fmt.Sprintf("Updated scan: %v", scanName),
			Contents:         fmt.Sprintf("The scan named %v, which you have access to, was just updated. Scan ID is: %v", scanName, scanId),
			From:             "Data Importer",
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       path.Join(n.envRootURL, "?q="+scanId),
			Meta:             map[string]string{},
		},
	}

	n.sendNotificationToObjectUsers(notifMsg, scanId)
}

func (n *NotificationSender) SysNotifyScanChanged(scanId string) {
	wsSysNotify := &protos.SysNotificationUpd{
		Reason:  protos.SysNotifyReason_SNR_SCAN,
		ScanIds: []string{scanId},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyNewScanImage(scanName string, scanId string, imageName string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          fmt.Sprintf("New image added to scan: %v", scanName),
			Contents:         fmt.Sprintf("A new image named %v was added to scan: %v (id: %v)", imageName, scanName, scanId),
			From:             "Data Importer",
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       path.Join(n.envRootURL, "?q="+scanId+"&image="+imageName),
			Meta:             map[string]string{},
		},
	}

	n.sendNotificationToObjectUsers(notifMsg, scanId)
}

func (n *NotificationSender) SysNotifyScanImagesChanged(scanIds []string) {
	wsSysNotify := &protos.SysNotificationUpd{
		Reason:  protos.SysNotifyReason_SNR_IMAGE,
		ScanIds: scanIds,
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyNewQuant(uploaded bool, quantId string, quantName string, status string, scanName string, scanId string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          fmt.Sprintf("Quantification %v has completed with status: %v", quantName, status),
			Contents:         fmt.Sprintf("A quantification named %v (id: %v) has completed with status %v. This quantification is for the scan named: %v", quantName, quantId, status, scanName),
			From:             "Data Importer",
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       path.Join(n.envRootURL, "?q="+scanId+"&quant="+quantId),
			Meta:             map[string]string{},
		},
	}

	n.sendNotificationToObjectUsers(notifMsg, quantId)
}

func (n *NotificationSender) SysNotifyQuantChanged(quantId string) {
	wsSysNotify := &protos.SysNotificationUpd{
		Reason:  protos.SysNotifyReason_SNR_QUANT,
		QuantId: quantId,
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyObjectShared(objectType string, objectId string, objectName, sharerName string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          fmt.Sprintf("%v was just shared", objectType),
			Contents:         fmt.Sprintf("An object of type %v named %v was just shared by %v", objectType, objectName, sharerName),
			From:             "PIXLISE back-end",
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       "",
			Meta:             map[string]string{},
		},
	}

	n.sendNotificationToObjectUsers(notifMsg, objectId)
}

func (n *NotificationSender) NotifyUserGroupMessage(subject string, message string, groupId string, groupName string, sender string) {
	notifMsg := &protos.UserNotificationUpd{
		Notification: &protos.UserNotification{
			Subject:          subject,
			Contents:         fmt.Sprintf("%v\nThis message was sent by %v to group %v", message, sender, groupName),
			From:             sender,
			TimeStampUnixSec: uint32(n.timestamper.GetTimeNowSec()),
			ActionLink:       "",
			Meta:             map[string]string{},
		},
	}

	userIds, err := wsHelpers.GetUserIdsForGroup([]string{groupId}, n.db)
	if err != nil {
		n.log.Errorf("Failed to get user ids for group: %v. Error: %v", groupId, err)
		return
	}

	n.sendNotification(subject, notifMsg, userIds)
}
