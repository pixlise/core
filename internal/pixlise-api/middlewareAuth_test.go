// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"fmt"
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
	var a authMiddleWareData
	a.routePermissionsRequired = map[string]string{
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
