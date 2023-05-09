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

package zenodo

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pixlise/core/v3/core/expressions/expressions"
	"github.com/pixlise/core/v3/core/expressions/modules"
	zenodoModels "github.com/pixlise/core/v3/core/expressions/zenodo-models"
)

func createEmptyDeposition(zenodoURI string, accessToken string) (*zenodoModels.ZenodoDepositionResponse, error) {
	emptyResponse := zenodoModels.ZenodoDepositionResponse{}

	depositionsURL := zenodoURI + "/api/deposit/depositions?access_token=" + accessToken

	resp, err := http.Post(depositionsURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return &emptyResponse, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &emptyResponse, err
	}

	response := zenodoModels.ZenodoDepositionResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return &emptyResponse, err
	}

	return &response, nil
}

func uploadFileContentsToZenodo(deposition zenodoModels.ZenodoDepositionResponse, filename string, contents *bytes.Buffer, accessToken string) (*zenodoModels.ZenodoFileUploadResponse, error) {
	emptyResponse := zenodoModels.ZenodoFileUploadResponse{}

	uploadUrl := deposition.Links.Bucket + "/" + filename + "?access_token=" + accessToken
	putReq, err := http.NewRequest("PUT", uploadUrl, contents)
	if err != nil {
		return &emptyResponse, err
	}

	putReq.Header.Set("Content-Type", "application/octet-stream")

	putResponse, err := http.DefaultClient.Do(putReq)
	if err != nil {
		return &emptyResponse, err
	}

	defer putResponse.Body.Close()
	putBody, err := ioutil.ReadAll(putResponse.Body)
	if err != nil {
		return &emptyResponse, err
	}

	fileUploadResponse := zenodoModels.ZenodoFileUploadResponse{}
	err = json.Unmarshal(putBody, &fileUploadResponse)
	if err != nil {
		return &emptyResponse, err
	}

	return &fileUploadResponse, nil
}

func uploadModuleToZenodo(deposition zenodoModels.ZenodoDepositionResponse, module modules.DataModuleSpecificVersionWire, accessToken string) (*zenodoModels.ZenodoFileUploadResponse, error) {
	zenodoResponse := zenodoModels.ZenodoFileUploadResponse{}

	filename := module.DataModule.ID + ".json"
	jsonContents, err := json.Marshal(module)
	if err != nil {
		return &zenodoResponse, err
	}

	fileUploadResponse, err := uploadFileContentsToZenodo(deposition, filename, bytes.NewBuffer([]byte(jsonContents)), accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	return fileUploadResponse, nil
}

func uploadExpressionZipToZenodo(deposition zenodoModels.ZenodoDepositionResponse, filename string, zipFile []byte, accessToken string) (*zenodoModels.ZenodoFileUploadResponse, error) {
	zenodoResponse := zenodoModels.ZenodoFileUploadResponse{}

	fileUploadResponse, err := uploadFileContentsToZenodo(deposition, filename, bytes.NewBuffer([]byte(zipFile)), accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	return fileUploadResponse, nil
}

func uploadMetadataToDeposition(deposition zenodoModels.ZenodoDepositionResponse, metadata map[string]interface{}, accessToken string) (*zenodoModels.ZenodoMetaResponse, error) {
	emptyResponse := zenodoModels.ZenodoMetaResponse{}

	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return &emptyResponse, err
	}

	depositionMetaURL := deposition.Links.LatestDraft + "?access_token=" + accessToken
	depositionMetaReq, err := http.NewRequest("PUT", depositionMetaURL, bytes.NewBuffer([]byte(metadataJson)))
	if err != nil {
		return &emptyResponse, err
	}

	depositionMetaReq.Header.Set("Content-Type", "application/json")
	metaResponse, err := http.DefaultClient.Do(depositionMetaReq)
	if err != nil {
		return &emptyResponse, err
	}

	defer metaResponse.Body.Close()

	metaObject, err := ioutil.ReadAll(metaResponse.Body)
	if err != nil {
		return &emptyResponse, err
	}

	metaResponseObj := zenodoModels.ZenodoMetaResponse{}
	err = json.Unmarshal(metaObject, &metaResponseObj)
	if err != nil {
		return &emptyResponse, err
	}

	return &metaResponseObj, nil
}

func addMetadataToDeposition(deposition zenodoModels.ZenodoDepositionResponse, doiMetadata zenodoModels.DOIMetadata, accessToken string) (*zenodoModels.ZenodoMetaResponse, error) {
	// API fails if any empty keys are included, so we need to remove them

	metadata := map[string]interface{}{
		"metadata": map[string]interface{}{
			"title":       doiMetadata.Title,
			"upload_type": "software",
			"description": doiMetadata.Description,
		},
	}

	if doiMetadata.Creators != nil && len(doiMetadata.Creators) > 0 {
		metadata["metadata"].(map[string]interface{})["creators"] = []map[string]interface{}{}

		for _, creator := range doiMetadata.Creators {
			creatorMap := map[string]interface{}{
				"name": creator.Name,
			}

			if creator.Affiliation != "" {
				creatorMap["affiliation"] = creator.Affiliation
			}

			if creator.Orcid != "" {
				creatorMap["orcid"] = creator.Orcid
			}

			metadata["metadata"].(map[string]interface{})["creators"] = append(metadata["metadata"].(map[string]interface{})["creators"].([]map[string]interface{}), creatorMap)
		}
	}

	if doiMetadata.Keywords != "" {
		metadata["metadata"].(map[string]interface{})["keywords"] = strings.Split(doiMetadata.Keywords, ",")
	}

	if doiMetadata.Notes != "" {
		metadata["metadata"].(map[string]interface{})["notes"] = doiMetadata.Notes
	}

	if doiMetadata.RelatedIdentifiers != nil && len(doiMetadata.RelatedIdentifiers) > 0 {
		metadata["metadata"].(map[string]interface{})["related_identifiers"] = []map[string]interface{}{}

		for _, relatedIdentifier := range doiMetadata.RelatedIdentifiers {
			relatedID := map[string]interface{}{
				"identifier": relatedIdentifier.Identifier,
				"relation":   relatedIdentifier.Relation,
			}
			metadata["metadata"].(map[string]interface{})["related_identifiers"] = append(metadata["metadata"].(map[string]interface{})["related_identifiers"].([]map[string]interface{}), relatedID)
		}
	}

	if doiMetadata.Contributors != nil && len(doiMetadata.Contributors) > 0 {
		metadata["metadata"].(map[string]interface{})["contributors"] = []map[string]interface{}{}

		for _, contributor := range doiMetadata.Contributors {
			contributorMap := map[string]interface{}{
				"name": contributor.Name,
			}

			if contributor.Affiliation != "" {
				contributorMap["affiliation"] = contributor.Affiliation
			}

			if contributor.Type != "" {
				contributorMap["type"] = contributor.Type
			}

			if contributor.Orcid != "" {
				contributorMap["orcid"] = contributor.Orcid
			}

			metadata["metadata"].(map[string]interface{})["contributors"] = append(metadata["metadata"].(map[string]interface{})["contributors"].([]map[string]interface{}), contributorMap)
		}
	}

	if doiMetadata.References != "" {
		metadata["metadata"].(map[string]interface{})["references"] = strings.Split(doiMetadata.References, ",")
	}

	if doiMetadata.Version != "" {
		metadata["metadata"].(map[string]interface{})["version"] = doiMetadata.Version
	}

	return uploadMetadataToDeposition(deposition, metadata, accessToken)
}

func publishDeposition(deposition zenodoModels.ZenodoDepositionResponse, accessToken string) (*zenodoModels.ZenodoPublishResponse, error) {
	publishURL := deposition.Links.Publish + "?access_token=" + accessToken
	publishReq, err := http.NewRequest(http.MethodPost, publishURL, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, err
	}

	publishReq.Header.Set("Content-Type", "application/json")
	publishResponse, err := http.DefaultClient.Do(publishReq)
	if err != nil {
		return nil, err
	}

	defer publishResponse.Body.Close()

	publishBody, err := ioutil.ReadAll(publishResponse.Body)
	if err != nil {
		return nil, err
	}

	zenodoResponse := zenodoModels.ZenodoPublishResponse{}
	err = json.Unmarshal(publishBody, &zenodoResponse)
	if err != nil {
		return nil, err
	}

	return &zenodoResponse, nil
}

func PublishModuleToZenodo(module modules.DataModuleSpecificVersionWire) (*zenodoModels.ZenodoPublishResponse, error) {
	zenodoResponse := zenodoModels.ZenodoPublishResponse{}

	accessToken, foundAccessToken := os.LookupEnv("ZENODO_ACCESS_TOKEN")
	if !foundAccessToken {
		return &zenodoResponse, errors.New("ZENODO_ACCESS_TOKEN not found")
	}

	zenodoURI, foundZenodoURI := os.LookupEnv("ZENODO_URI")
	if !foundZenodoURI {
		return &zenodoResponse, errors.New("ZENODO_URI not found")
	}

	deposition, err := createEmptyDeposition(zenodoURI, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	_, err = uploadModuleToZenodo(*deposition, module, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	_, err = addMetadataToDeposition(*deposition, module.Version.DOIMetadata, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	publishResponse, err := publishDeposition(*deposition, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	zenodoResponse = *publishResponse
	return &zenodoResponse, nil
}

func PublishExpressionZipToZenodo(expression expressions.DataExpression, zipFile []byte) (*zenodoModels.ZenodoPublishResponse, error) {
	zenodoResponse := zenodoModels.ZenodoPublishResponse{}

	accessToken, foundAccessToken := os.LookupEnv("ZENODO_ACCESS_TOKEN")
	if !foundAccessToken {
		return &zenodoResponse, errors.New("ZENODO_ACCESS_TOKEN not found")
	}

	zenodoURI, foundZenodoURI := os.LookupEnv("ZENODO_URI")
	if !foundZenodoURI {
		return &zenodoResponse, errors.New("ZENODO_URI not found")
	}

	deposition, err := createEmptyDeposition(zenodoURI, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	filename := expression.ID + ".zip"
	_, err = uploadExpressionZipToZenodo(*deposition, filename, zipFile, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	_, err = addMetadataToDeposition(*deposition, expression.DOIMetadata, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	publishResponse, err := publishDeposition(*deposition, accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	zenodoResponse = *publishResponse
	return &zenodoResponse, nil
}
