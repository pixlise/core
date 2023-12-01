package notificationSender

import (
	"fmt"
	"path"

	"github.com/pixlise/core/v3/api/ws"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/timestamper"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type NotificationSender struct {
	db          *mongo.Database
	timestamper timestamper.ITimeStamper // So we can mock time.Now()
	log         logger.ILogger
	envRootURL  string
	ws          *ws.WSHandler
}

func MakeNotificationSender(db *mongo.Database, timestamper timestamper.ITimeStamper, log logger.ILogger, envRootURL string, ws *ws.WSHandler) *NotificationSender {
	return &NotificationSender{
		db:          db,
		timestamper: timestamper,
		log:         log,
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

func (n *NotificationSender) NotifyQuantComplete(quantId string, quantName string, status string, scanName string, scanId string) {
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

	n.sendNotification(notifMsg, userIds)
}

func (n *NotificationSender) sendNotificationToObjectUsers(notifMsg *protos.UserNotificationUpd, objectId string) {
	userIds, err := wsHelpers.FindUserIdsFor(objectId, n.db)
	if err != nil {
		n.log.Errorf("Failed to get user ids for object: %v. Error: %v", objectId, err)
		return
	}

	n.sendNotification(notifMsg, userIds)
}

func (n *NotificationSender) sendNotification(notifMsg *protos.UserNotificationUpd, userIds []string) {
	// Loop through each user, if we have them connected, notify directly, otherwise email
	sessions, noSessionUserIds := n.ws.GetSessionForUsersIfExists(userIds)
	for _, session := range sessions {
		msg := &protos.WSMessage{Contents: &protos.WSMessage_UserNotificationUpd{UserNotificationUpd: notifMsg}}
		wsHelpers.SendForSession(session, msg)
	}

	// Email the rest
	for _, emailUserId := range noSessionUserIds {
		n.sendEmail(notifMsg.Notification, emailUserId)
	}
}

func (n *NotificationSender) sendEmail(notif *protos.UserNotification, userId string) {
	// Find the email address
	user, err := wsHelpers.GetDBUser(userId, n.db)
	if err != nil {
		n.log.Errorf("sendEmail: Failed to get user info for user id: %v. Error: %v", userId, err)
		return
	}

	if user.Info == nil {
		n.log.Errorf("sendEmail: Get user info got nil info item for user id: %v", userId)
		return
	}

	if len(user.Info.Email) <= 0 {
		n.log.Errorf("sendEmail: User %v had empty email address", userId)
		return
	}

	unsub := "You can change your notification subscriptions if you log into PIXLISE and click on the user icon"
	text := fmt.Sprintf(`Hi %v,

%v

%v.`, user.Info.Name, notif.Contents, unsub)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>%v</title>
</head>
<body>
<h3>Hi %v</h3>
<p>%v</p>
<p>%v</p>
</body>
</html>
`, notif.Subject, user.Info.Name, notif.Contents, unsub)

	awsutil.SESSendEmail(user.Info.Email, "UTF-8", text, html, notif.Subject, "info@mail.pixlise.org", []string{}, []string{})
}
