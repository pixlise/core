package idleTime

import (
	"fmt"

	"github.com/pixlise/core/v4/core/timestamper"
)

func Example_idleTime_Test() {
	ts := timestamper.MockTimeNowStamper{QueuedTimeStamps: []int64{
		1779919000, // startup
		1779919000, // 0 nodes
		1779919002, // 1
		1779919010, // 3
		1779919050, // 2
		1779919054, // 0
		1779919060, // 0
		1779919066, // 0

		1779919500, // startup
		1779919500, // 0
		1779919509, // 0
		1779919511, // 0
	}}
	i := MakeIdleTimeChecker(10, &ts)

	fmt.Println("Test job starting soon after startup")
	nodes := []int{0, 1, 3, 2, 0, 0, 0}
	for c, jobs := range nodes {
		fmt.Printf("%v: jobs %v, isIdle: %v\n", c, jobs, i.HasIdleTimeExpired(uint(jobs)))
	}

	fmt.Println("Test no job coming")
	i = MakeIdleTimeChecker(10, &ts)

	nodes = []int{0, 0, 0}
	for c, jobs := range nodes {
		fmt.Printf("%v: jobs %v, isIdle: %v\n", c, jobs, i.HasIdleTimeExpired(uint(jobs)))
	}

	// Output:
	// Test job starting soon after startup
	// 0: jobs 0, isIdle: false
	// 1: jobs 1, isIdle: false
	// 2: jobs 3, isIdle: false
	// 3: jobs 2, isIdle: false
	// 4: jobs 0, isIdle: false
	// 5: jobs 0, isIdle: false
	// 6: jobs 0, isIdle: true
	// Test no job coming
	// 0: jobs 0, isIdle: false
	// 1: jobs 0, isIdle: false
	// 2: jobs 0, isIdle: true
}
