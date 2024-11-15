package pixlem

import (
	"fmt"
	"time"
)

func Example_timeToTestSol() {
	t, err := time.Parse("2006-002T15:04:05", "2019-180T11:02:31")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2024-318T13:42:47")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2017-068T13:42:47")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2017-001T00:00:00")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2017-000T00:00:00")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2020-400T00:00:00")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	t, err = time.Parse("2006-002T15:04:05", "2014-020T00:00:00")
	fmt.Printf("%v|%v\n", err, timeToTestSol(t))

	// Output:
	// <nil>|C180
	// <nil>|H318
	// <nil>|A068
	// <nil>|A001
	// parsing time "2017-000T00:00:00": day-of-year out of range|????
	// parsing time "2020-400T00:00:00": day-of-year out of range|????
	// <nil>|????
}
