package wsHelpers

import (
	"context"
	"errors"
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"google.golang.org/protobuf/proto"
)

func SendNotification(userIds []string, notification *protos.UserNotification, hctx HandlerContext) error {
	if len(userIds) <= 0 {
		return errors.New("attempted to send notification with empty user list")
	}

	// Set some basic fields
	notification.TimeStampUnixSec = uint64(hctx.Svcs.TimeStamper.GetTimeNowSec())

	// Create an update message to send
	wsMsg := protos.WSMessage{
		Contents: &protos.WSMessage_UserNotificationUpd{
			UserNotificationUpd: &protos.UserNotificationUpd{
				Notification: notification,
			},
		},
	}

	bytes, err := proto.Marshal(&wsMsg)
	if err != nil {
		return err
	}

	// We broadcast to any users who are in the list of users...
	// Track which users we broadcasted to - the others would NOT have received the
	// broadcast (likely they're currently offline, so we will write to DB)
	userIdsToSaveDB := map[string]bool{}

	// Assume saving for evereyone
	for _, usrId := range userIds {
		userIdsToSaveDB[usrId] = true
	}

	// Work out who we're broadcasting to and remove them from save list
	sessions, err := hctx.Melody.Sessions()
	if err != nil {
		return err
	}

	// Loop through and find which users are NOT in a session, so we know who to write to DB for
	for _, sess := range sessions {
		usr, err := GetSessionUser(sess)
		if err != nil {
			hctx.Svcs.Log.Errorf("Failed to determine session user id when broadcasting: %v", sess)
		}

		if utils.ItemInSlice(usr.User.Id, userIds) && usr.NotificationSubscribed {
			// We'll broadcast to this one, so don't save to DB
			userIdsToSaveDB[usr.User.Id] = false
		}
	}

	// Save to DB for users we didn't see in a session above
	for usrId, save := range userIdsToSaveDB {
		if save {
			//fmt.Printf("SendNotification user %v saving to DB\n", usrId)

			_err := saveForUser(usrId, notification, hctx)
			if _err != nil {
				// We have no other recourse at this stage, just print it
				hctx.Svcs.Log.Errorf("Error while saving notification for user id: %v. Error was: %v", usrId, _err)
			}
		}
	}

	callback := func(sess *melody.Session) bool {
		usr, err := GetSessionUser(sess)
		if err != nil {
			hctx.Svcs.Log.Errorf("Failed to determine session user id when broadcasting: %v", sess)
			return false // not sending here
		}

		if save, ok := userIdsToSaveDB[usr.User.Id]; ok {
			// if NOT saving, we want to return true, so it gets broadcast here
			if !save {
				fmt.Printf("Broadcasting to user: %v\n", usr.User.Id)
				return true
			}
		}

		// User is not in the list of save vs send, so don't send
		return false
	}

	err = hctx.Melody.BroadcastBinaryFilter(bytes, callback)
	if err != nil {
		// We have no other recourse at this stage, just print it
		hctx.Svcs.Log.Errorf("Error while sending notification broadcast for %+v. Error was: %v", notification, err)
	}

	return nil
}

func saveForUser(userId string, notification *protos.UserNotification, hctx HandlerContext) error {
	// Make a copy which has the user id set
	toSave := &protos.UserNotificationDB{
		DestUserId:   userId,
		Notification: notification,
	}
	_, err := hctx.Svcs.MongoDB.Collection(dbCollections.NotificationsName).InsertOne(context.TODO(), toSave)
	return err
}

//func GetUsersToNotify(requiredPermissions []string, )
