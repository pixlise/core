package utils

import (
	"fmt"
	"time"

	"github.com/pixlise/core/v4/core/timestamper"
)

func Example_rateLimiter() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Hard limit")
		}
	}()

	ts := timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567000, 1234567000, 1234567001, 1234567003, 1234567009, 12345670010, 12345670012, 12345670013, 1234567017, 1234567025, 1234567030, 1234567030, 1234567031, 1234567032},
	}
	r := MakeRateLimiter(&ts, 3, 5, 10, 5)

	runLoop(r, len(ts.QueuedTimeStamps))

	// Output:
	// No limit
	// No limit
	// No limit
	// Rate limiting 5 sec...
	// Soft limit
	// Rate limiting 5 sec...
	// Soft limit
	// No limit
	// No limit
	// No limit
	// Rate limiting 5 sec...
	// Soft limit
	// Rate limiting 5 sec...
	// Soft limit
	// Rate limiting 5 sec...
	// Soft limit
	// Hard limit
}

func runLoop(r *RateLimiter, iterations int) {
	for c := 0; c < iterations; c++ {
		startMs := time.Now().UnixMilli()
		r.CheckRateLimit()
		endMs := time.Now().UnixMilli()

		if endMs-startMs > 4000 {
			fmt.Println("Soft limit")
		} else {
			fmt.Println("No limit")
		}
	}
}
