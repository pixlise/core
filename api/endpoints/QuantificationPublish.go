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

package endpoints

import (
	"github.com/pixlise/core/api/handlers"
	quant "github.com/pixlise/core/core/quantModel"
)

func quantificationPublish(params handlers.ApiHandlerParams) (interface{}, error) {
	// Get the ids invovled
	//datasetID := params.PathParams[datasetIdentifier]
	//jobID := params.PathParams[idIdentifier]
	//datasetID := params.pathParams[datasetIdentifier]
	//jobID := params.pathParams[idIdentifier]
	//return nil, errors.New("Not implemented yet!")
	dataset := params.PathParams[datasetIdentifier]
	jobid := params.PathParams[idIdentifier]
	config := quant.PublisherConfig{
		KubernetesLocation:      params.Svcs.Config.KubernetesLocation,
		QuantDestinationPackage: params.Svcs.Config.QuantDestinationPackage,
		QuantObjectType:         params.Svcs.Config.QuantObjectType,
		PosterImage:             params.Svcs.Config.PosterImage,
		DatasetsBucket:          params.Svcs.Config.DatasetsBucket,
		EnvironmentName:         params.Svcs.Config.EnvironmentName,
		Kubeconfig:              params.Svcs.Config.KubeConfig,
		UsersBucket:             params.Svcs.Config.UsersBucket,
	}
	err := quant.PublishQuant(params.Svcs.FS, config, params.UserInfo,
		params.Svcs.Log, dataset, jobid, params.Svcs.Notifications)
	return nil, err
}
