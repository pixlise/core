package jobmanager

import (
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/services"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func completePythonScript(jg *protos.JobGroupConfig, lastJobStatus *protos.JobStatus, session *melody.Session, svcs *services.APIServices) (*protos.JobStatus, error) {
	now := svcs.TimeStamper.GetTimeNowSec()
	return &protos.JobStatus{
		JobId:                 lastJobStatus.JobId,
		JobItemId:             lastJobStatus.JobItemId,
		JobType:               lastJobStatus.JobType,
		Status:                protos.JobStatus_COMPLETE,
		Message:               fmt.Sprintf("Python script %v ran in %vsec", jg.JobName, now-int64(lastJobStatus.StartUnixTimeSec)),
		StartUnixTimeSec:      lastJobStatus.StartUnixTimeSec,
		LastUpdateUnixTimeSec: uint32(now),
		EndUnixTimeSec:        uint32(now),
		//OutputFilePath:   memoKey,
		OtherLogFiles:   []string{},
		Name:            jg.JobName,
		Elements:        jg.ElementList,
		RequestorUserId: jg.RequestorUserId,
	}, nil
}
