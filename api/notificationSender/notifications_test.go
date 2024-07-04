package notificationSender

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func Example_notificationDBSave() {
	db := wstestlib.GetDB()
	ctx := context.TODO()

	// Clear relevant collections
	db.Collection(dbCollections.NotificationsName).Drop(ctx)

	notif := &protos.Notification{
		Id:               "ajm5d5bjs4vq7bkc-auth0|5de45d85ca40070f421a3a34",
		DestUserId:       "auth0|5de45d85ca40070f421a3a34",
		Subject:          "Quantification Peters combined v4 test quant has completed with status: Complete",
		Contents:         "A quantification named Peters combined v4 test quant (id: quant-wqp66sayhowj41ej) has completed with status Complete. This quantification is for the scan named: Castle Geyser",
		From:             "Data Importer",
		TimeStampUnixSec: 1710831761,
		ActionLink:       "?q=393871873&quant=quant-wqp66sayhowj41ej",
		NotificationType: protos.NotificationType_NT_USER_MESSAGE,
	}

	n := MakeNotificationSender("abc123", db, nil, nil, &logger.StdOutLoggerForTest{}, "unittest", nil, nil)
	fmt.Printf("Notification write to empty DB: %v\n", n.saveNotificationToDB("notif123", "destuser123", notif))
	fmt.Printf("Notification overwrite: %v", n.saveNotificationToDB("notif123", "destuser123", notif))

	// Output:
	// Notification write to empty DB: <nil>
	// Notification overwrite: <nil>
}
