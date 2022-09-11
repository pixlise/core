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
	"github.com/pixlise/core/v2/api/handlers"
	quant "github.com/pixlise/core/v2/core/quantModel"
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
