package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleSendUserNotificationReq(req *protos.SendUserNotificationReq, hctx wsHelpers.HandlerContext) (*protos.SendUserNotificationResp, error) {
	// Send from a user, need to define destination, could be group/user ids?
	// Probably messaging, subject+content, can send as email if not connected?
	// Think of load balance issue with multiple APIs running
	// Think of deep linking case, eg data party, people sending out a link to what they're viewing, again group based broadcasting
	// Should be able to specify if sending to active sessions vs storing in DB for later user retrieval

	// Automated ones:
	// New scan, sent to the group the scan belongs to, not just active session - also sent as email
	//
	// Scan updated, sent to the group the scan belongs to, not just active session - also sent as email.
	//       From field we could filter on, eg Jesper spamming. Potentially UI asks user what changed
	//
	// Quant complete, sent to user who requested, email if not active session (could be sent on quant success AND error/other exit clause)
	//       NOTE: quant progress should be sent out as part of job messaging, independent of this, not emailed, etc.
	//
	// Something shared (quant, roi, expr, workspace/collection etc) (sent to group who was shared to), say who shared it, maybe include an id, include deep link
	//
	// Custom notification - someone could type a notification and send to a user/group. From field should say who it's from so receivers could filter it

	return nil, errors.New("HandleSendUserNotificationReq not implemented yet")
}

func HandleUserNotificationReq(req *protos.UserNotificationReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationResp, error) {
	// Triggers a "subscription" to receive updates containing notifications for the session user
	// Could implement a "silent" mode, specify param in request, tell API to not send notifications for a certain period

	return nil, errors.New("HandleUserNotificationReq not implemented yet")
}
