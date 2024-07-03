package notificationSender

import (
	"context"
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/singleinstance"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/proto"
)

// NOTE: This must be in sync with the client! It saves user notification settings with topic strings
//
//	and these have to match what the UI is setting for the topic field or the user setting lookup
//	will fail and we won't send out that notification
//
// At time of writing, these are defined in:
//
//	pixlise-ui\client\src\app\modules\settings\models\notification.model.ts
var NOTIF_TOPIC_SCAN_NEW = "New Dataset Available"
var NOTIF_TOPIC_SCAN_UPDATED = "Dataset Updated"
var NOTIF_TOPIC_QUANT_COMPLETE = "Qunatification Complete"
var NOTIF_TOPIC_IMAGE_NEW = "New Image For Dataset"
var NOTIF_TOPIC_OBJECT_SHARED = "Object Shared"

func (n *NotificationSender) sendSysNotification(sysNotification *protos.NotificationUpd) {
	wsSysNotify := protos.WSMessage{
		Contents: &protos.WSMessage_NotificationUpd{
			NotificationUpd: sysNotification,
		},
	}

	bytes, err := proto.Marshal(&wsSysNotify)
	if err == nil {
		n.melody.BroadcastBinary(bytes)
	}

	/* For reference, we also had another implementation using broadcasting with filters:

	callback := func(sess *melody.Session) bool {
		usr, err := GetSessionUser(sess)
		if err != nil {
			hctx.Svcs.Log.Errorf("Failed to determine session user id when broadcasting: %v", sess)
			return false // not sending here
		}

		if we need to send {
			return true
		}

		// User is not in the list of save vs send, so don't send
		return false
	}

	err = hctx.Melody.BroadcastBinaryFilter(bytes, callback)
	*/
}

func (n *NotificationSender) sendNotificationToObjectUsers(topic string, notifMsg *protos.NotificationUpd, objectId string) {
	userIds, err := wsHelpers.FindUserIdsFor(objectId, n.db)
	if err != nil {
		n.log.Errorf("Failed to get user ids for object: %v. Error: %v", objectId, err)
		return
	}

	n.sendNotification(objectId, topic, notifMsg, userIds)
}

// SourceId must be an id that is unique across API instances so we can decide on one instance to send emails from!
func (n *NotificationSender) sendNotification(sourceId string, topicId string, notifMsg *protos.NotificationUpd, userIds []string) {
	if len(userIds) <= 0 {
		n.log.Errorf("No users to send notification \"%v\" to!", notifMsg.Notification.Subject)
		return
	}

	// Ensure the notification has a unique ID from here, because we're sending/storing them and may need dismissability
	origId := notifMsg.Notification.Id
	if len(origId) <= 0 {
		origId = n.idgen.GenObjectID()
	}

	// Ensure other fields are set too
	notifMsg.Notification.TimeStampUnixSec = uint32(n.timestamper.GetTimeNowSec())

	// Retrieve notification settings for each user, save the user IDs in separate lists for UI vs email notifications
	uiNotificationUsers := []string{}
	emailNotificationUsers := []string{}

	for _, userId := range userIds {
		// Write it to DB if needed
		err := n.saveNotificationToDB(origId, userId, notifMsg.Notification)
		if err != nil {
			n.log.Errorf("Failed to save notification to DB for user: %v. Error: \"%v\". Notification was: %+v", userId, err, notifMsg.Notification)
		}

		user, err := wsHelpers.GetDBUser(userId, n.db)
		if err != nil {
			n.log.Errorf("Failed to retrieve user %v when sending notification. Nothing sent. Error was: %v", userId, err)
		} else {
			// Check what kind of notification this user requires. NOTE if the topic is empty, we assume it's UI only
			if len(topicId) <= 0 {
				uiNotificationUsers = append(uiNotificationUsers, userId)
			} else {
				method := user.NotificationSettings.TopicSettings[topicId]

				if method == protos.NotificationMethod_NOTIF_BOTH || method == protos.NotificationMethod_NOTIF_EMAIL {
					emailNotificationUsers = append(emailNotificationUsers, userId)
				}

				if method == protos.NotificationMethod_NOTIF_BOTH || method == protos.NotificationMethod_NOTIF_UI {
					uiNotificationUsers = append(uiNotificationUsers, userId)
				}

				/* Removed because it's really not that helpful but spams logs heaps because lots of users don't have notifications on!
				if method == protos.NotificationMethod_NOTIF_NONE {
					n.log.Debugf("Skipping notification of topic: %v to user %v because they have this topic turned off", topicId, userId)
				}
				*/
			}
		}
	}

	// Send UI notifications to whoevere is connected
	// NOTE: This won't work reliably in prod at present because users may be connected to another instance of the API and wouldn't
	//       receive this. There's a card for building a cross-API notification situation here.
	if len(uiNotificationUsers) > 0 {
		sessions, _ := n.ws.GetSessionForUsersIfExists(uiNotificationUsers)
		for _, session := range sessions {
			// Send it with a unique ID for this user
			sessUser, err := wsHelpers.GetSessionUser(session)
			if err == nil {
				notifMsg.Notification.Id = origId + "-" + sessUser.User.Id
				msg := &protos.WSMessage{Contents: &protos.WSMessage_NotificationUpd{NotificationUpd: notifMsg}}

				n.log.Infof("Sending UI notification: %v, with id: %v to user: %v", notifMsg.Notification.Subject, notifMsg.Notification.Id, sessUser.User.Id)
				wsHelpers.SendForSession(session, msg)
			} else {
				n.log.Errorf("Error: %v - notification not sent!", err)
			}
		}
	}

	// Send emails, but only from ONE instance of our API!
	if len(emailNotificationUsers) > 0 {
		singleinstance.HandleOnce(sourceId, n.instanceId, func(sourceId string) {
			// ID is not relevant here...
			notifMsg.Notification.Id = ""

			// NOTE: At this point we have no way to exclude emails for those sessions we have already sent
			//       web socket notifications to because multiple API instances have done the above job, but
			//       email sending is being done by one instance which doesn't have a list of all sessions
			//       connected to all APIs, so here we email all interested parties.
			for _, emailUserId := range emailNotificationUsers {
				n.sendEmail(notifMsg.Notification, emailUserId)
			}
		}, n.db, n.timestamper, n.log)
	}
}

func (n *NotificationSender) sendEmail(notif *protos.Notification, userId string) {
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

	actionLink := ""
	actionLinkHTML := ""
	if len(notif.ActionLink) > 0 {
		link := path.Join(n.envRootURL, notif.ActionLink)
		actionLink = fmt.Sprintf("\nPIXLISE Link: %v", link)
		actionLinkHTML = fmt.Sprintf("\n<p>PIXLISE Link: <a href=\"%v\">%v</a></p>", link, link)
	}

	unsub := "You can change your notification subscriptions if you log into PIXLISE and click on the user icon"
	text := fmt.Sprintf(`Hi %v,

%v%v

%v.`, user.Info.Name, notif.Contents, actionLink, unsub)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>%v</title>
</head>
<body>
<h3>Hi %v</h3>
<p>%v</p>
<p>%v</p>%v
</body>
</html>
`, notif.Subject, user.Info.Name, notif.Contents, actionLinkHTML, unsub)

	n.log.Infof("Sending email notification: %v, to user: %v, email: %v", notif.Subject, user.Info.Id, user.Info.Email)
	awsutil.SESSendEmail(user.Info.Email, "UTF-8", text, html, notif.Subject, "info@mail.pixlise.org", []string{}, []string{})
}

func (n *NotificationSender) saveNotificationToDB(notifId string, destUserId string, notification *protos.Notification) error {
	toSave := &protos.Notification{
		DestUserId: destUserId,

		Id:               notifId + "-" + destUserId,
		DestUserGroupId:  notification.DestUserGroupId,
		MaxSecToExpiry:   notification.MaxSecToExpiry,
		Subject:          notification.Subject,
		Contents:         notification.Contents,
		From:             notification.From,
		TimeStampUnixSec: notification.TimeStampUnixSec,
		ActionLink:       notification.ActionLink,
		NotificationType: notification.NotificationType,
		ScanIds:          notification.ScanIds,
		ImageName:        notification.ImageName,
		QuantId:          notification.QuantId,
	}

	// Make a copy which has the user id set
	filter := bson.D{{"id", toSave.Id}}
	_, err := n.db.Collection(dbCollections.NotificationsName).ReplaceOne(context.TODO(), filter, toSave, options.Replace().SetUpsert(true))
	return err
}
