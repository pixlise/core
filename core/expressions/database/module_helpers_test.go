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

package expressionDB

import (
	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/logger"
)

func makeMockSvcs(idGen services.IDGenerator) services.APIServices {
	cfg := config.APIConfig{}

	return services.APIServices{
		Config: cfg,
		Log:    &logger.NullLogger{},
		SNS:    &awsutil.MockSNS{},
		IDGen:  idGen,
	}
}
