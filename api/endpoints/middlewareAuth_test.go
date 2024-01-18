// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package endpoints

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_isMatch() {
	// Matches
	fmt.Println(isMatch("GET/roi", "GET/roi"))
	fmt.Println(isMatch("GET/roi/", "GET/roi"))
	fmt.Println(isMatch("GET/roi", "GET/roi/"))
	fmt.Println(isMatch("GET/roi/", "GET/roi/"))

	fmt.Println(isMatch("PUT/roi/123", "PUT/roi/{cat}"))
	fmt.Println(isMatch("PUT/roi/123/", "PUT/roi/{cat}"))
	fmt.Println(isMatch("PUT/roi/123", "PUT/roi/{cat}/"))
	fmt.Println(isMatch("PUT/roi/123/", "PUT/roi/{cat}/"))

	fmt.Println(isMatch("POST/roi/123/999", "POST/roi/{cat}/{id}"))
	fmt.Println(isMatch("POST/roi/123/999/", "POST/roi/{cat}/{id}"))
	fmt.Println(isMatch("POST/roi/123/999", "POST/roi/{cat}/{id}/"))
	fmt.Println(isMatch("POST/roi/123/999/", "POST/roi/{cat}/{id}/"))

	fmt.Println(isMatch("DELETE/roi/123/path", "DELETE/roi/{cat}/path"))
	fmt.Println(isMatch("DELETE/roi/123/path/", "DELETE/roi/{cat}/path"))
	fmt.Println(isMatch("DELETE/roi/123/path", "DELETE/roi/{cat}/path/"))
	fmt.Println(isMatch("DELETE/roi/123/path/", "DELETE/roi/{cat}/path/"))

	fmt.Println(isMatch("GET/roi/123/path/999", "GET/roi/{cat}/path/{id}"))
	fmt.Println(isMatch("GET/roi/123/path/999/", "GET/roi/{cat}/path/{id}"))
	fmt.Println(isMatch("GET/roi/123/path/999", "GET/roi/{cat}/path/{id}/"))
	fmt.Println(isMatch("GET/roi/123/path/999/", "GET/roi/{cat}/path/{id}/"))

	// Fails
	fmt.Println(isMatch("/roi", "/roi"))
	fmt.Println(isMatch("SAVE/roi", "SAVE/roi"))
	fmt.Println(isMatch("GET/", "GET/roi"))
	fmt.Println(isMatch("GET/roi", "GET/roi/{cat}"))
	fmt.Println(isMatch("GET/roi/", "GET/roi/{cat}"))
	fmt.Println(isMatch("GET/roi/999", "GET/roi/{cat}/{id}"))
	fmt.Println(isMatch("GET/roi/999/", "GET/roi/{cat}/{id}"))
	fmt.Println(isMatch("GET/roi/999/", "GET/roi/{cat}/path/{id}"))
	fmt.Println(isMatch("GET/roi/999/path", "GET/roi/{cat}/path/{id}"))
	fmt.Println(isMatch("GET/roi/999/path/", "GET/roi/{cat}/path/{id}"))
	fmt.Println(isMatch("GET/roi/999/pathy/11", "GET/roi/{cat}/path/{id}"))

	// Output:
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// true
	// false
	// false
	// false
	// false
	// false
	// false
	// false
	// false
	// false
	// false
	// false
}

func Example_getPermissionsForURI() {
	var a AuthMiddleWareData
	a.Logger = &logger.StdOutLogger{}
	a.RoutePermissionsRequired = map[string]string{
		"GET/the/{id}/something": "root3",
		"GET/the/path":           "root1",
		"POST/the/path":          "root1a",
		"PUT/the/path/something": "root2",
	}

	fmt.Println(a.getPermissionsForURI("GET", "/the/121/something"))
	fmt.Println(a.getPermissionsForURI("GET", "/the/path"))
	fmt.Println(a.getPermissionsForURI("PUT", "/the/path/something"))
	fmt.Println(a.getPermissionsForURI("GET", "/the/121/path"))
	fmt.Println(a.getPermissionsForURI("GET", "/the/121/path/another"))
	fmt.Println(a.getPermissionsForURI("GET", "/"))
	fmt.Println(a.getPermissionsForURI("GET", "/obj"))
	fmt.Println(a.getPermissionsForURI("POST", "/the/path"))
	fmt.Println(a.getPermissionsForURI("PUT", "/the/121/something"))

	// Output:
	// root3 <nil>
	// root1 <nil>
	// root2 <nil>
	//  Permissions not defined for route: GET /the/121/path
	//  Permissions not defined for route: GET /the/121/path/another
	//  Permissions not defined for route: GET /
	//  Permissions not defined for route: GET /obj
	// root1a <nil>
	//  Permissions not defined for route: PUT /the/121/something

}
