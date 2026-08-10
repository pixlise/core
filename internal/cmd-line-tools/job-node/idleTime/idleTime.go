package idleTime

import (
	"github.com/pixlise/core/v4/core/timestamper"
)

// Checks if the idle time is past the threshold allowed. In other words
// we quit if we'e been sitting idle for more than given seconds
type IdleTimeChecker struct {
	lastJobCount         int
	lastJobFinishUnixSec int64
	allowedIdleTimeSec   int64
	ts                   timestamper.ITimeStamper
}

func MakeIdleTimeChecker(allowedIdleTimeSec int64, ts timestamper.ITimeStamper) IdleTimeChecker {
	// To start out, set the lastJobFinishUnixSec time to now, so if we sit idle for the idle sec
	// time we quit regardless of if there were jobs run. We don't want to sit here for 3 days
	// waiting for a job!
	nowUnixSec := ts.GetTimeNowSec()
	return IdleTimeChecker{0, nowUnixSec, allowedIdleTimeSec, ts}
}

func (i *IdleTimeChecker) HasIdleTimeExpired(activeJobs uint) bool {
	nowUnixSec := i.ts.GetTimeNowSec()

	if i.lastJobCount > 0 && activeJobs == 0 {
		// The last job has quit, remember this time
		i.lastJobFinishUnixSec = nowUnixSec
	}

	// Remember this job count, so we can detect when the last job quits
	i.lastJobCount = int(activeJobs)

	// If we have no active jobs, and the last job finished at longer than the threshold, we say the time has expired
	return activeJobs == 0 && i.lastJobFinishUnixSec > 0 && nowUnixSec-i.lastJobFinishUnixSec > i.allowedIdleTimeSec
}
