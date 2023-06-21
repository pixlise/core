package main

import (
	"fmt"
	"sort"
	"strings"
)

func printRoutePermissions(routePermissions map[string]string) {
	// Gather keys
	paths := []string{}
	longestPath := 0
	for k := range routePermissions {
		pathStart := strings.Index(k, "/")
		method := k[0:pathStart]
		path := k[pathStart:]

		// Store it so it's sortable but we can split it later
		paths = append(paths, fmt.Sprintf("%v|%v|%v", path, method, k))

		pathLen := len(path)
		if pathLen > longestPath {
			longestPath = pathLen
		}
	}
	sort.Strings(paths)

	// Print
	fmt.Println("Route Permissions:")
	fmtString := fmt.Sprintf("%%-7v%%-%vv -> %%v\n", longestPath)

	for _, path := range paths {
		// Make it more presentable
		bits := strings.Split(path, "|")
		path := bits[0]
		method := bits[1]
		query := bits[2]

		fmt.Printf(fmtString, method, path, routePermissions[query])
	}
}
