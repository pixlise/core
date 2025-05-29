package notificationSender

import (
	"fmt"
	"path"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/ws"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/idgen"
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
	idgen       idgen.IDGenerator
}

func MakeNotificationSender(instanceId string, db *mongo.Database, idgen idgen.IDGenerator, timestamper timestamper.ITimeStamper, log logger.ILogger, envRootURL string, ws *ws.WSHandler, melody *melody.Melody) *NotificationSender {
	return &NotificationSender{
		instanceId:  instanceId,
		db:          db,
		timestamper: timestamper,
		log:         log,
		ws:          ws,
		melody:      melody,
		idgen:       idgen,
		envRootURL:  envRootURL,
	}
}

func (n *NotificationSender) NotifyNewScan(scanName string, scanId string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_USER_MESSAGE,
			Subject:          fmt.Sprintf("New scan imported: %v", scanName),
			Contents:         fmt.Sprintf("A new scan named %v was just imported. Scan ID is: %v.", scanName, scanId),
			From:             "Data Importer",
			ActionLink:       fmt.Sprintf("analysis?scan_id=%v", scanId),
		},
	}

	n.sendNotificationToObjectUsers(NOTIF_TOPIC_SCAN_NEW, notifMsg, scanId)
}

func (n *NotificationSender) NotifyUpdatedScan(scanName string, scanId string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_USER_MESSAGE,
			Subject:          fmt.Sprintf("Updated scan: %v", scanName),
			Contents:         fmt.Sprintf("The scan named %v, which you have access to, was just updated. Scan ID is: %v.", scanName, scanId),
			From:             "Data Importer",
			ActionLink:       fmt.Sprintf("analysis?scan_id=%v", scanId),
		},
	}

	n.sendNotificationToObjectUsers(NOTIF_TOPIC_SCAN_UPDATED, notifMsg, scanId)
}

func (n *NotificationSender) SysNotifyScanChanged(scanId string) {
	wsSysNotify := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_SYS_DATA_CHANGED,
			ScanIds:          []string{scanId},
		},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) SysNotifyROIChanged(roiId string) {
	wsSysNotify := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_SYS_DATA_CHANGED,
			RoiId:            roiId,
		},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) SysNotifyMapChanged(mapId string) {
	wsSysNotify := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_SYS_DATA_CHANGED,
			MapId:            mapId,
		},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyNewScanImage(scanName string, scanId string, imageName string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_USER_MESSAGE,
			Subject:          fmt.Sprintf("New image added to scan: %v", scanName),
			Contents:         fmt.Sprintf("A new image named %v was added to scan: %v (id: %v)", path.Base(imageName), scanName, scanId),
			From:             "Data Importer",
			ActionLink:       fmt.Sprintf("analysis?scan_id=%v&image=%v", scanId, imageName),
		},
	}

	n.sendNotificationToObjectUsers(NOTIF_TOPIC_IMAGE_NEW, notifMsg, scanId)
}

func (n *NotificationSender) SysNotifyScanImagesChanged(imageName string, scanIds []string) {
	wsSysNotify := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_SYS_DATA_CHANGED,
			ImageName:        imageName,
			ScanIds:          scanIds,
		},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyNewQuant(uploaded bool, quantId string, quantName string, status string, scanName string, scanId string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_USER_MESSAGE,
			Subject:          fmt.Sprintf("Quantification %v has completed with status: %v", quantName, status),
			Contents:         fmt.Sprintf("A quantification named %v (id: %v) has completed with status %v. This quantification is for the scan named: %v", quantName, quantId, status, scanName),
			From:             "Data Importer",
			ActionLink:       fmt.Sprintf("analysis?scan_id=%v&quant=%v", scanId, quantId),
		},
	}

	n.sendNotificationToObjectUsers(NOTIF_TOPIC_QUANT_COMPLETE, notifMsg, quantId)
}

func (n *NotificationSender) SysNotifyQuantChanged(quantId string) {
	wsSysNotify := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_SYS_DATA_CHANGED,
			QuantId:          quantId,
		},
	}

	n.sendSysNotification(wsSysNotify)
}

func (n *NotificationSender) NotifyObjectShared(objectType string, objectId string, objectName, sharerName string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: protos.NotificationType_NT_USER_MESSAGE,
			Subject:          fmt.Sprintf("%v was just shared", objectType),
			Contents:         fmt.Sprintf("An object of type %v named %v was just shared by %v", objectType, objectName, sharerName),
			From:             "PIXLISE back-end",
			ActionLink:       "",
		},
	}

	n.sendNotificationToObjectUsers(NOTIF_TOPIC_OBJECT_SHARED, notifMsg, objectId)
}

func (n *NotificationSender) NotifyUserGroupMessage(subject string, message string, notificationType protos.NotificationType, actionLink string, groupId string, groupName string, sender string) {
	userIds, err := wsHelpers.GetUserIdsForGroup([]string{groupId}, n.db)
	if err != nil {
		n.log.Errorf("Failed to get user ids for group: %v. Error: %v", groupId, err)
		return
	}

	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: notificationType,
			Subject:          subject,
			Contents:         fmt.Sprintf("%v\nThis message was sent by %v to group %v", message, sender, groupName),
			From:             sender,
			ActionLink:       actionLink,
		},
	}

	n.sendNotification(subject, "", notifMsg, userIds)
}

func (n *NotificationSender) NotifyUserMessage(subject string, message string, notificationType protos.NotificationType, actionLink string, requestorUserId string, destUserIds []string, sender string) {
	notifMsg := &protos.NotificationUpd{
		Notification: &protos.Notification{
			NotificationType: notificationType,
			Subject:          subject,
			Contents:         fmt.Sprintf("%v\nThis message was sent by %v", message, sender),
			From:             sender,
			ActionLink:       actionLink,
			RequestorUserId:  requestorUserId,
		},
	}

	n.sendNotification(subject, "", notifMsg, destUserIds)
}
