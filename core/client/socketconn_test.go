package client

import (
	"fmt"
)

func Example_normaliseURLAndPath() {
	u, p := normaliseURLAndPath("https://www.pixlise.org", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("https://www.pixlise.org", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	u, p = normaliseURLAndPath("https://www.pixlise.org/", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("https://www.pixlise.org/", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	u, p = normaliseURLAndPath("www.pixlise.org/", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("www.pixlise.org/", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	u, p = normaliseURLAndPath("https://www.pixlise.org/api", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("https://www.pixlise.org/api", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	u, p = normaliseURLAndPath("https://www.pixlise.org/api/", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("https://www.pixlise.org/api/", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	u, p = normaliseURLAndPath("www.pixlise.org/api/", "/ws-connect")
	fmt.Printf("%v|%v\n", u, p)
	u, p = normaliseURLAndPath("www.pixlise.org/api/", "ws-connect")
	fmt.Printf("%v|%v\n", u, p)

	// ur, e := url.Parse("https://www.pixlise.org/api/ws-connect")
	// fmt.Printf("%v|%v|%v|%v\n", e, ur.Scheme, ur.Host, ur.Path)

	// ur, e = url.Parse("www.pixlise.org/api/ws-connect")
	// fmt.Printf("%v|%v|%v|%v\n", e, ur.Scheme, ur.Host, ur.Path)

	// Output:
	// https://www.pixlise.org|/ws-connect
	// https://www.pixlise.org|/ws-connect
	// https://www.pixlise.org|/ws-connect
	// https://www.pixlise.org|/ws-connect
	// www.pixlise.org|/ws-connect
	// www.pixlise.org|/ws-connect
	// https://www.pixlise.org|/api/ws-connect
	// https://www.pixlise.org|/api/ws-connect
	// https://www.pixlise.org|/api/ws-connect
	// https://www.pixlise.org|/api/ws-connect
	// www.pixlise.org|/api/ws-connect
	// www.pixlise.org|/api/ws-connect
}
