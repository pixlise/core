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

package handlers

import (
	"errors"
	"net/http"

	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/pixlUser"
)

// If it's a share function, use this. Enforces that method is only POST
type ApiSharingHandler struct {
	*services.APIServices
	Share ApiHandlerFunc
}

func (h ApiSharingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	var err error
	if r.Method != "POST" {
		err = errors.New("Share must be POST")
	} else {
		var userInfo pixlUser.UserInfo
		userInfo, err = h.APIServices.JWTReader.GetUserInfo(r)

		if err == nil {
			resp, errShare := h.Share(ApiHandlerParams{h.APIServices, userInfo, pathParams, r})
			err = errShare
			if err == nil {
				api.ToJSON(w, resp)
			}
		}
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
