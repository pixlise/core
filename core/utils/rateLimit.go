package utils

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v4/core/timestamper"
)

type RateLimiter struct {
	requestTimestamps []int64
	timestamper       timestamper.ITimeStamper
	softLimitInWindow int
	hardLimitInWindow int
	timeWindowSec     int64
	softLimitSleepSec int
}

func MakeRateLimiter(timestamper timestamper.ITimeStamper, softLimit int, hardLimit int, timeWindowSec int64, softLimitSleepSec int) *RateLimiter {
	return &RateLimiter{
		requestTimestamps: []int64{},
		timestamper:       timestamper,
		softLimitInWindow: softLimit,
		hardLimitInWindow: hardLimit,
		timeWindowSec:     timeWindowSec,
		softLimitSleepSec: softLimitSleepSec,
	}
}

func (r *RateLimiter) CheckRateLimit() {
	now := r.timestamper.GetTimeNowSec()

	oldest := now - r.timeWindowSec

	// Clear too old
	validTimestamps := []int64{}
	for _, ts := range r.requestTimestamps {
		if ts >= oldest {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Add ours
	r.requestTimestamps = append(validTimestamps, now)

	// Check if we need to limit
	if len(r.requestTimestamps) > r.hardLimitInWindow {
		panic("Message hard rate limit exceeded")
	}

	if len(r.requestTimestamps) > r.softLimitInWindow {
		fmt.Printf("Rate limiting %v sec...\n", r.softLimitSleepSec)
		time.Sleep(time.Duration(r.softLimitSleepSec) * time.Second)
	}
}
