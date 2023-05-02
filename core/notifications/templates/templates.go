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

// Text templates for various types of notifications including HTML vs simple text versions
package textTemplates

var datasetUpdatedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ .TemplateMap.datasourcename}} Datasource Updated</title>
</head>
<body>
<h3>Hi {{ .ContentMap.Name }}</h3>
<p>The datasource {{ .TemplateMap.datasourcename}} has been updated.</p>
<p>{{ .TemplateMap.extrainfohtml }}</p>
</body>
</html>
`

var datasetUpdatedTXT = `Hi {{ .ContentMap.Name }}

The {{ .TemplateMap.datasourcename}} datasource has been updated.
{{ .TemplateMap.extrainfotxt }}
`

var datasetUpdatedSMS = `Hi {{ .ContentMap.Name }}

The {{ .TemplateMap.datasourcename}} datasource has been updated.
`

var datasetUpdatedUI = `The {{ .TemplateMap.datasourcename}} has been updated.`

var newDatasetAvailableHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>New Datasource Available</title>
</head>
<body>
<h3>Hi {{ .ContentMap.Name }}</h3>
<p>New datasource {{ .TemplateMap.datasourcename}} is available.</p>
<p>{{ .TemplateMap.extrainfohtml }}</p>
</body>
</html>
`

var newDatasetAvailableTXT = `Hi {{ .ContentMap.Name }}

New datasource {{ .TemplateMap.datasourcename}} is available.
{{ .TemplateMap.extrainfotxt }}.
`

var newDatasetAvailableSMS = `Hi {{ .ContentMap.Name }}

New datasource {{ .TemplateMap.datasourcename}} is available.
`

var newDatasetAvailableUI = `New datasource {{ .TemplateMap.datasourcename}} is available.`

var quantProcessingFailedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Quantification processing failed</title>
</head>
<body>
<h3>Hi {{ .ContentMap.Config.Name }}</h3>
<p>We're very sorry the quantification {{ .TemplateMap.quantname }} has failed to quantify, please try again or contact support for more help.</p>
</body>
</html>
`

var quantProcessingFailedTXT = `Hi {{ .ContentMap.Config.Name }}

We're very sorry the quantification {{ .TemplateMap.quantname }} has failed to quantify, please try again or contact support for more help.`

var quantProcessingFailedSMS = `Hi {{ .ContentMap.Config.Name }}

We're very sorry the quantification {{ .TemplateMap.quantname }} has failed to quantify, please try again or contact support for more help.`

var quantProcessingFailedUI = `We're very sorry the quantification  {{ .TemplateMap.quantname }} has failed to quantify, please try again or contact support for more help.`

var testDatasetAvailableHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>New Datasource Available</title>
</head>
<body>
<h3>Hi {{ .ContentMap.Name }}</h3>
<p>New datasource {{ .TemplateMap.datasourcename}} is available.</p>
</body>
</html>`

var testDatasetAvailableTXT = `Hi {{ .ContentMap.Name }}

New datasource {{ .TemplateMap.datasourcename}} is available.`

var testDatasetAvailableSMS = `Hi {{ .ContentMap.Name }}

New datasource {{ .TemplateMap.datasourcename}} is available.
`

var testDatasetAvailableUI = `New datasource {{ .TemplateMap.datasourcename}} is available.`

var userQuantCompleteHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Quantification processing complete</title>
</head>
<body>
<h3>Hi {{ .ContentMap.Config.Name }}</h3>
<p>The processing of your latest quantification, {{ .TemplateMap.quantname }} , is complete.</p>
<p>Please login to <a href="https://www.pixlise.org">Pixlise</a> to see the results.</p>
</body>
</html>`

var userQuantCompleteTXT = `Hi {{ .ContentMap.Config.Name }}

The processing of your latest quantification, {{ .TemplateMap.quantname }} , is complete.

Please login to Pixlise to see the results.`

var userQuantCompleteSMS = `Hi {{ .ContentMap.Config.Name }}

The processing of your latest quantification, {{ .TemplateMap.quantname }} , is complete.`

var userQuantCompleteUI = `The processing of your latest quantification, {{ .TemplateMap.quantname }} , is complete.`

var quantpublishedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"><title>Quantification Published</title></meta>
</head>
<h3>Hi {{ .ContentMap.Config.Name }}</h3>
<p>The publishing of the following quantification {{ .TemplateMap.quantname }}, to datadrive is complete.</p>
</body>
</html>`

var quantpublishedTXT = `Hi {{ .ContentMap.Config.Name }}

The publishing of the following quantification {{ .TemplateMap.quantname }}, to datadrive is complete

Thanks,
Pixlise`

var quantpublishedSMS = `The publishing of the quantification {{ .TemplateMap.quantname }} is complete.`

var quantpublishedUI = `The publishing of the quantification {{ .TemplateMap.quantname }} is complete.`

func GetTemplates() map[string]string {
	var m = make(map[string]string)
	m["dataset-updated-HTML"] = datasetUpdatedHTML
	m["dataset-updated-TXT"] = datasetUpdatedTXT
	m["dataset-updated-UI"] = datasetUpdatedUI
	m["dataset-Updated-SMS"] = datasetUpdatedSMS

	m["new-dataset-available-HTML"] = newDatasetAvailableHTML
	m["new-dataset-available-TXT"] = newDatasetAvailableTXT
	m["new-dataset-available-SMS"] = newDatasetAvailableSMS
	m["new-dataset-available-UI"] = newDatasetAvailableUI

	m["quant-processing-failed-HTML"] = quantProcessingFailedHTML
	m["quant-processing-failed-TXT"] = quantProcessingFailedTXT
	m["quant-processing-failed-SMS"] = quantProcessingFailedSMS
	m["quant-processing-failed-UI"] = quantProcessingFailedUI

	m["test-dataset-available-HTML"] = testDatasetAvailableHTML
	m["test-dataset-available-TXT"] = testDatasetAvailableTXT
	m["test-dataset-available-SMS"] = testDatasetAvailableSMS
	m["test-dataset-available-UI"] = testDatasetAvailableUI

	m["user-quant-complete-HTML"] = userQuantCompleteHTML
	m["user-quant-complete-TXT"] = userQuantCompleteTXT
	m["user-quant-complete-UI"] = userQuantCompleteUI
	m["user-quant-complete-SMS"] = userQuantCompleteSMS

	m["quant-published-HTML"] = quantpublishedHTML
	m["quant-published-TXT"] = quantpublishedTXT
	m["quant-published-SMS"] = quantpublishedSMS
	m["quant-published-UI"] = quantpublishedUI

	return m
}
