package wsHandler

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func createEmptyDeposition(zenodoURI string, accessToken string) (*protos.ZenodoDepositionResponse, error) {
	emptyResponse := protos.ZenodoDepositionResponse{}

	depositionsURL := zenodoURI + "/api/deposit/depositions?access_token=" + accessToken

	resp, err := http.Post(depositionsURL, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return &emptyResponse, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &emptyResponse, err
	}

	response := protos.ZenodoDepositionResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return &emptyResponse, err
	}

	return &response, nil
}

func uploadFileContentsToZenodo(deposition *protos.ZenodoDepositionResponse, filename string, contents *bytes.Buffer, accessToken string) (*protos.ZenodoFileUploadResponse, error) {
	emptyResponse := protos.ZenodoFileUploadResponse{}

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
	putBody, err := io.ReadAll(putResponse.Body)
	if err != nil {
		return &emptyResponse, err
	}

	fileUploadResponse := protos.ZenodoFileUploadResponse{}
	err = json.Unmarshal(putBody, &fileUploadResponse)
	if err != nil {
		return &emptyResponse, err
	}

	return &fileUploadResponse, nil
}

func uploadModuleToZenodo(deposition *protos.ZenodoDepositionResponse, module *protos.DataModule, accessToken string) (*protos.ZenodoFileUploadResponse, error) {
	zenodoResponse := protos.ZenodoFileUploadResponse{}

	filename := module.Id + ".json"
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

func uploadExpressionZipToZenodo(deposition *protos.ZenodoDepositionResponse, filename string, zipFile string, accessToken string) (*protos.ZenodoFileUploadResponse, error) {
	zenodoResponse := protos.ZenodoFileUploadResponse{}

	fileUploadResponse, err := uploadFileContentsToZenodo(deposition, filename, bytes.NewBuffer([]byte(zipFile)), accessToken)
	if err != nil {
		return &zenodoResponse, err
	}

	return fileUploadResponse, nil
}

func uploadMetadataToDeposition(deposition *protos.ZenodoDepositionResponse, metadata map[string]interface{}, accessToken string) (*protos.ZenodoMetaResponse, error) {
	emptyResponse := protos.ZenodoMetaResponse{}

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

	metaObject, err := io.ReadAll(metaResponse.Body)
	if err != nil {
		return &emptyResponse, err
	}

	metaResponseObj := protos.ZenodoMetaResponse{}
	err = json.Unmarshal(metaObject, &metaResponseObj)
	if err != nil {
		return &emptyResponse, err
	}

	return &metaResponseObj, nil
}

func addMetadataToDeposition(deposition *protos.ZenodoDepositionResponse, doiMetadata *protos.DOIMetadata, accessToken string) (*protos.ZenodoMetaResponse, error) {
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

func publishDeposition(deposition *protos.ZenodoDepositionResponse, accessToken string) (*protos.ZenodoPublishResponse, error) {
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

	publishBody, err := io.ReadAll(publishResponse.Body)
	if err != nil {
		return nil, err
	}

	zenodoResponse := protos.ZenodoPublishResponse{}
	err = json.Unmarshal(publishBody, &zenodoResponse)
	if err != nil {
		return nil, err
	}

	return &zenodoResponse, nil
}

func PublishModuleToZenodo(module *protos.DataModule, metadata *protos.DOIMetadata, zenodoURI string, zenodoToken string) (*protos.ZenodoPublishResponse, error) {
	if zenodoURI == "" {
		return nil, errors.New("ZENODO_URI not found")
	}

	if zenodoToken == "" {
		return nil, errors.New("ZENODO_ACCESS_TOKEN not found")
	}

	deposition, err := createEmptyDeposition(zenodoURI, zenodoToken)
	if err != nil {
		return nil, err
	}

	_, err = uploadModuleToZenodo(deposition, module, zenodoToken)
	if err != nil {
		return nil, err
	}

	_, err = addMetadataToDeposition(deposition, metadata, zenodoToken)
	if err != nil {
		return nil, err
	}

	publishResponse, err := publishDeposition(deposition, zenodoToken)
	if err != nil {
		return nil, err
	}

	return publishResponse, nil
}

func PublishExpressionToZenodo(id string, output string, metadata *protos.DOIMetadata, mongo *mongo.Database, zenodoURI string, zenodoToken string) (*protos.ZenodoPublishResponse, error) {
	deposition, err := createEmptyDeposition(zenodoURI, zenodoToken)
	if err != nil {
		return nil, err
	}

	filename := id + ".zip"
	_, err = uploadExpressionZipToZenodo(deposition, filename, output, zenodoToken)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": id}
	opts := options.Find()
	_, err = mongo.Collection(dbCollections.ExpressionsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	_, err = addMetadataToDeposition(deposition, metadata, zenodoToken)
	if err != nil {
		return nil, err
	}

	publishResponse, err := publishDeposition(deposition, zenodoToken)
	if err != nil {
		return nil, err
	}

	return publishResponse, nil
}

func HandlePublishExpressionToZenodoReq(req *protos.PublishExpressionToZenodoReq, hctx wsHelpers.HandlerContext) ([]*protos.PublishExpressionToZenodoResp, error) {
	if hctx.Svcs.Config.EnvironmentName == "unittest" || hctx.Svcs.Config.EnvironmentName == "local" {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	zenodoURI := hctx.Svcs.Config.ZenodoURI
	zenodoToken := hctx.Svcs.Config.ZenodoAccessToken

	if zenodoURI == "" {
		return nil, errors.New("ZENODO_URI not found")
	}

	if zenodoToken == "" {
		return nil, errors.New("ZENODO_ACCESS_TOKEN not found")
	}

	publishResponse, err := PublishExpressionToZenodo(req.Id, req.Output, req.Metadata, hctx.Svcs.MongoDB, zenodoURI, zenodoToken)
	if err != nil {
		return nil, err
	}

	metadata := protos.DOIMetadata{
		Id:                 req.Id,
		Title:              publishResponse.Title,
		Description:        publishResponse.Metadata.Description,
		Creators:           req.Metadata.Creators,
		Keywords:           req.Metadata.Keywords,
		Notes:              req.Metadata.Notes,
		RelatedIdentifiers: req.Metadata.RelatedIdentifiers,
		Contributors:       req.Metadata.Contributors,
		References:         req.Metadata.References,
		Version:            req.Metadata.Version,
		Doi:                publishResponse.Doi,
		DoiBadge:           publishResponse.Links.Badge,
		DoiLink:            publishResponse.Links.Doi,
	}

	// Write to DOI collection
	_, err = hctx.Svcs.MongoDB.Collection(dbCollections.DOIName).InsertOne(context.TODO(), &metadata)
	if err != nil {
		return nil, err
	}

	return []*protos.PublishExpressionToZenodoResp{&protos.PublishExpressionToZenodoResp{
		Doi: &metadata,
	}}, nil
}

func HandleZenodoDOIGetReq(req *protos.ZenodoDOIGetReq, hctx wsHelpers.HandlerContext) ([]*protos.ZenodoDOIGetResp, error) {
	metadata := &protos.DOIMetadata{}
	err := hctx.Svcs.MongoDB.Collection(dbCollections.DOIName).FindOne(context.TODO(), bson.D{{Key: "_id", Value: req.Id}}).Decode(&metadata)
	if err != nil {
		return nil, err
	}

	return []*protos.ZenodoDOIGetResp{&protos.ZenodoDOIGetResp{
		Doi: metadata,
	}}, nil
}
