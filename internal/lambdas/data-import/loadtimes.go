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
	"time"

	"github.com/pixlise/core/core/fileaccess"
)

type Loaded struct {
	LastLoaded []LastLoaded `json:"last_loaded"`
}
type LastLoaded struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
}

func saveLoadtime(name string, loads Loaded, fs fileaccess.FileAccess) error {
	var newloads []LastLoaded
	for _, l := range loads.LastLoaded {
		if l.Name == name {
			l.Timestamp = time.Now()
		}
		newloads = append(newloads, l)
	}
	var l = Loaded{newloads}

	return fs.WriteJSONNoIndent(getConfigBucket(), "configs/lastloaded.json", l)
}

func lookupLoadtime(name string, fs fileaccess.FileAccess) (Loaded, bool) {
	var loads Loaded
	err := fs.ReadJSON(getConfigBucket(), "configs/lastloaded.json", &loads, false)
	if err != nil {
		// REFACTOR: Return an error? What if this fails, is it bad? Should we use the "return empty if not found" flag above?
		fmt.Println(err)
	}
	for _, r := range loads.LastLoaded {
		if r.Name == name {
			if time.Now().Sub(r.Timestamp).Hours() < 1 {
				return loads, true
			} else {
				return loads, false
			}
		}
	}
	return loads, false
}
