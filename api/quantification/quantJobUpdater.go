package quantification

import (
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuantJobUpdater struct {
	params  *protos.QuantCreateParams
	session *melody.Session
	//melody   *melody.Melody
	notifier services.INotifier
	db       *mongo.Database
}

func MakeQuantJobUpdater(
	params *protos.QuantCreateParams,
	session *melody.Session,
	//melody *melody.Melody,
	notifier services.INotifier,
	db *mongo.Database,
) QuantJobUpdater {
	return QuantJobUpdater{
		params:  params,
		session: session,
		//melody:   melody,
		notifier: notifier,
		db:       db,
	}
}

func (i *QuantJobUpdater) SendQuantJobUpdate(status *protos.JobStatus) {
	if i.session != nil {
		wsUpd := protos.WSMessage{
			Contents: &protos.WSMessage_QuantCreateUpd{
				QuantCreateUpd: &protos.QuantCreateUpd{
					Status: status,
				},
			},
		}

		wsHelpers.SendForSession(i.session, &wsUpd)
	}

	// If the job has completed, notify out
	if status.Status == protos.JobStatus_COMPLETE {
		scan, err := scan.ReadScanItem(i.params.ScanId, i.db)
		if err != nil {
			fmt.Errorf("sendQuantJobUpdate for completed job failed to read scan: %v", i.params.ScanId)
			return
		}

		i.notifier.NotifyNewQuant(false, status.JobItemId, i.params.Name, "Complete", scan.Title, i.params.ScanId)
	}
}
