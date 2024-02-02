package notificationSender

import (
	"fmt"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/singleinstance"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func (n *NotificationSender) sendNotificationToObjectUsers(notifMsg *protos.UserNotificationUpd, objectId string) {
	userIds, err := wsHelpers.FindUserIdsFor(objectId, n.db)
	if err != nil {
		n.log.Errorf("Failed to get user ids for object: %v. Error: %v", objectId, err)
		return
	}

	n.sendNotification(objectId, notifMsg, userIds)
}

// SourceId must be an id that is unique across API instances so we can decide on one instance to send emails from!
func (n *NotificationSender) sendNotification(sourceId string, notifMsg *protos.UserNotificationUpd, userIds []string) {
	// Loop through each user, if we have them connected, notify directly, otherwise email
	sessions, _ := n.ws.GetSessionForUsersIfExists(userIds)
	for _, session := range sessions {
		msg := &protos.WSMessage{Contents: &protos.WSMessage_UserNotificationUpd{UserNotificationUpd: notifMsg}}
		wsHelpers.SendForSession(session, msg)
	}

	// Email the rest, but only from ONE instance of our API!
	singleinstance.HandleOnce(sourceId, n.instanceId, func(sourceId string) {
		// NOTE: At this point we have no way to exclude emails for those sessions we have already sent
		//       web socket notifications to because multiple API instances have done the above job, but
		//       email sending is being done by one instance which doesn't have a list of all sessions
		//       connected to all APIs, so here we email all interested parties.
		for _, emailUserId := range userIds {
			n.sendEmail(notifMsg.Notification, emailUserId)
		}
	}, n.db, n.timestamper, n.log)
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
