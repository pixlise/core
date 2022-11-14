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

package notifications

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/pixlise/core/v2/core/logger"
	textTemplates "github.com/pixlise/core/v2/core/notifications/templates"
)

const (
	charSet = "UTF-8"
)

// TemplateContents - Structure for template injection
type TemplateContents struct {
	ContentMap  UserStruct
	TemplateMap map[string]interface{}
}

func generateEmailContent(log logger.ILogger, subscriber UserStruct, templateName string, templateInput map[string]interface{}, format string) (string, error) {
	t := textTemplates.GetTemplates()
	var templates = template.Must(template.New(templateName).Parse(t[templateName+"-"+format]))

	inv := TemplateContents{ContentMap: subscriber, TemplateMap: templateInput}
	var tpl bytes.Buffer

	log.Debugf("Executing Template: %v, %v", templateName, inv)
	err := templates.ExecuteTemplate(&tpl, templateName, inv)
	if err != nil {
		errToReturn := fmt.Errorf("Failed to generate template: %v", err)
		log.Errorf("%v", errToReturn)
		return "", err
	}
	result := tpl.String()
	return result, nil
}
