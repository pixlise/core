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

import "github.com/pixlise/core/core/fileaccess"

type configfile struct {
	Name       string `json:"name"`
	Detector   string `json:"detector"`
	Group      string `json:"group"`
	UpdateType string `json:"updateType"`
}

func getConfigFile() (configfile, error) {
	localFS := fileaccess.FSAccess{}

	var config configfile
	err := localFS.ReadJSON(localUnzipPath, "config.json", &config, false)
	return config, err
}

// TODO: This should probably take a configfile struct as a parameter...
func computeName() (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}
	return config.Name, nil
}

// TODO: This should probably take a configfile struct as a parameter...
func customDetector(sol string) (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.Detector != "" {
		// Return a custom detector string.
		return config.Detector, nil
	} else if sol[0] >= '0' && sol[0] <= '9' {
		// Usual Sol number and no custom string, don't override.
		return "", nil
	} else if sol[0] == 'D' || sol[0] == 'C' {
		return "", nil
	} else {
		// Sol starts with a character, non-standard, use the EM detector.
		return "PIXL-EM-E2E", nil
	}
}

// TODO: This should probably take a configfile struct as a parameter...
func customGroup(detector string) (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.Group != "" {
		// Return a custom detector string.
		return config.Group, nil
	} else if detector == "PIXL-EM-E2E" {
		return "PIXL-EM", nil
	} else {
		return "PIXL-FM", nil
	}
}

// TODO: This should probably take a configfile struct as a parameter...
func overrideUpdateType() (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.UpdateType != "" {
		// Return a custom detector string.
		return config.UpdateType, nil
	} else {
		return "full", nil
	}
}
