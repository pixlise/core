package quantification

import (
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type QuantJobUpdater struct {
	params  *protos.QuantCreateParams
	session *melody.Session
	//melody   *melody.Melody
	notifier    services.INotifier
	db          *mongo.Database
	fs          fileaccess.FileAccess
	usersBucket string
}

func MakeQuantJobUpdater(
	params *protos.QuantCreateParams,
	session *melody.Session,
	//melody *melody.Melody,
	notifier services.INotifier,
	db *mongo.Database,
	fs fileaccess.FileAccess,
	usersBucket string,
) QuantJobUpdater {
	return QuantJobUpdater{
		params:  params,
		session: session,
		//melody:   melody,
		notifier:    notifier,
		db:          db,
		fs:          fs,
		usersBucket: usersBucket,
	}
}

// We send updates for quant jobs (that are long running, have names, stick around) this way...
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

		i.notifier.NotifyNewQuant(false, status.JobId, i.params.Name, "Complete", scan.Title, i.params.ScanId)
		i.notifier.SysNotifyQuantChanged(status.JobId)
	}
}

// We send updates for ephemeral quant jobs (that are short running) this way...
func (i *QuantJobUpdater) SendEphemeralQuantJobUpdate(status *protos.JobStatus) {
	// We only send out an update for a completed job...
	if status.Status == protos.JobStatus_COMPLETE && i.session != nil {
		// We send out the result data, as opposed to a status
		userOutputFilePath := filepaths.GetUserLastPiquantOutputPath(status.RequestorUserId, i.params.ScanId, i.params.Command, filepaths.QuantLastOutputFileName+".csv")
		bytes, err := i.fs.ReadObject(i.usersBucket, userOutputFilePath)
		if err != nil {
			fmt.Errorf("PIQUANT job ephermal status failed to find output for: %v", status.JobId)
		}

		wsUpd := protos.WSMessage{
			Contents: &protos.WSMessage_QuantCreateUpd{
				QuantCreateUpd: &protos.QuantCreateUpd{
					// Need to include status info too otherwise receiver doesn't know what fit this is for
					Status:     status,
					ResultData: bytes,
				},
			},
		}

		wsHelpers.SendForSession(i.session, &wsUpd)
	}
}
