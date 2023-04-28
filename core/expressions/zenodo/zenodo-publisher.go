package zenodo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/pixlise/core/v3/core/expressions/modules"
)

type ZenodoPublishResponse struct {
	ConceptDOI   string `json:"conceptdoi"`
	ConceptRecID string `json:"conceptrecid"`
	Created      string `json:"created"`
	DOI          string `json:"doi"`
	DOIURL       string `json:"doi_url"`

	Files []struct {
		Checksum string `json:"checksum"`
		Filename string `json:"filename"`
		Filesize int    `json:"filesize"`
		ID       string `json:"id"`

		Links struct {
			Download string `json:"download"`
			Self     string `json:"self"`
		} `json:"links"`
	} `json:"files"`

	ID int `json:"id"`

	Links struct {
		Badge        string `json:"badge"`
		Bucket       string `json:"bucket"`
		ConceptBadge string `json:"conceptbadge"`
		ConceptDOI   string `json:"conceptdoi"`
		DOI          string `json:"doi"`
		Latest       string `json:"latest"`
		LatestHTML   string `json:"latest_html"`
		Record       string `json:"record"`
		RecordHTML   string `json:"record_html"`
	} `json:"links"`

	Metadata struct {
		AccessRight string `json:"access_right"`

		Communities []struct {
			Identifier string `json:"identifier"`
		} `json:"communities"`

		Creators []struct {
			Name string `json:"name"`
		} `json:"creators"`

		Description string `json:"description"`
		DOI         string `json:"doi"`
		License     string `json:"license"`

		PrereserveDOI struct {
			DOI   string `json:"doi"`
			RecID int    `json:"recid"`
		} `json:"prereserve_doi"`

		PublicationDate string `json:"publication_date"`
		Title           string `json:"title"`
		UploadType      string `json:"upload_type"`
	} `json:"metadata"`

	Modified string `json:"modified"`
	Owner    int    `json:"owner"`

	RecordID int    `json:"record_id"`
	State    string `json:"state"`

	Submitted bool   `json:"submitted"`
	Title     string `json:"title"`
}

type ZenodoDepositionMetadata struct {
	AccessRight string `json:"access_right"`

	Communities []struct {
		Identifier string `json:"identifier"`
	} `json:"communities"`

	Creators []struct {
		Name        string `json:"name"`
		Affiliation string `json:"affiliation"`
	} `json:"creators"`

	Description string `json:"description"`
	DOI         string `json:"doi"`
	License     string `json:"license"`

	PrereserveDOI struct {
		DOI   string `json:"doi"`
		RecID int    `json:"recid"`
	} `json:"prereserve_doi"`

	PublicationDate string `json:"publication_date"`
	Title           string `json:"title"`
	UploadType      string `json:"upload_type"`
}

type ZenodoMetaResponse struct {
	ConceptRecID string `json:"conceptrecid"`
	Created      string `json:"created"`
	DOI          string `json:"doi"`
	DOIURL       string `json:"doi_url"`

	Files []struct {
		Checksum string `json:"checksum"`
		Filename string `json:"filename"`
		Filesize int    `json:"filesize"`
		ID       string `json:"id"`

		Links struct {
			Download string `json:"download"`
			Self     string `json:"self"`
		} `json:"links"`
	} `json:"files"`

	ID int `json:"id"`

	Links struct {
		Badge        string `json:"badge"`
		Bucket       string `json:"bucket"`
		ConceptBadge string `json:"conceptbadge"`
		ConceptDOI   string `json:"conceptdoi"`
		DOI          string `json:"doi"`
		Latest       string `json:"latest"`
		LatestHTML   string `json:"latest_html"`
		Record       string `json:"record"`
		RecordHTML   string `json:"record_html"`
	} `json:"links"`

	Metadata ZenodoDepositionMetadata `json:"metadata"`

	Modified string `json:"modified"`
	Owner    int    `json:"owner"`

	RecordID int    `json:"record_id"`
	State    string `json:"state"`

	Submitted bool   `json:"submitted"`
	Title     string `json:"title"`
}

type ZenodoDepositionResponse struct {
	ConceptRecID string `json:"conceptrecid"`
	Created      string `json:"created"`

	Files []struct {
		Links struct {
			Download string `json:"download"`
		} `json:"links"`
	} `json:"files"`

	ID    int `json:"id"`
	Links struct {
		Bucket          string `json:"bucket"`
		Discard         string `json:"discard"`
		Edit            string `json:"edit"`
		Files           string `json:"files"`
		HTML            string `json:"html"`
		LatestDraft     string `json:"latest_draft"`
		LatestDraftHTML string `json:"latest_draft_html"`
		Publish         string `json:"publish"`
		Self            string `json:"self"`
	}

	Meta struct {
		PrereserveDOI struct {
			DOI   string `json:"doi"`
			RecID int    `json:"recid"`
		} `json:"prereserve_doi"`
	} `json:"metadata"`

	Owner     int    `json:"owner"`
	RecordID  int    `json:"record_id"`
	State     string `json:"state"`
	Submitted bool   `json:"submitted"`
	Title     string `json:"title"`
}

type ZenodoFileUploadResponse struct {
	Key       string `json:"key"`
	Mimetype  string `json:"mimetype"`
	Checksum  string `json:"checksum"`
	VersionID string `json:"version_id"`
	Size      int    `json:"size"`
	Created   string `json:"created"`
	Updated   string `json:"updated"`

	Links struct {
		Self    string `json:"self"`
		Version string `json:"version"`
		Uploads string `json:"uploads"`
	} `json:"links"`

	IsHead       bool `json:"is_head"`
	DeleteMarker bool `json:"delete_marker"`
}

func createEmptyDeposition(zenodoURI string, accessToken string) (*ZenodoDepositionResponse, error) {
	emptyResponse := ZenodoDepositionResponse{}

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

	response := ZenodoDepositionResponse{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return &emptyResponse, err
	}

	return &response, nil
}

func uploadFileContentsToZenodo(deposition ZenodoDepositionResponse, filename string, contents *bytes.Buffer, accessToken string) (*ZenodoFileUploadResponse, error) {
	emptyResponse := ZenodoFileUploadResponse{}

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

	fileUploadResponse := ZenodoFileUploadResponse{}
	err = json.Unmarshal(putBody, &fileUploadResponse)
	if err != nil {
		return &emptyResponse, err
	}

	return &fileUploadResponse, nil
}

func uploadModuleToZenodo(deposition ZenodoDepositionResponse, module modules.DataModuleSpecificVersionWire, accessToken string) (*ZenodoFileUploadResponse, error) {
	zenodoResponse := ZenodoFileUploadResponse{}

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

func addMetadataToDeposition(deposition ZenodoDepositionResponse, metadata ZenodoDepositionMetadata, accessToken string) (*ZenodoMetaResponse, error) {
	emptyResponse := ZenodoMetaResponse{}

	metadataJson, err := json.Marshal(map[string]interface{}{"metadata": metadata})
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

	metaResponseObj := ZenodoMetaResponse{}
	err = json.Unmarshal(metaObject, &metaResponseObj)
	if err != nil {
		return &emptyResponse, err
	}

	return &metaResponseObj, nil
}

func addModuleMetadataToDeposition(deposition ZenodoDepositionResponse, module modules.DataModuleSpecificVersionWire, accessToken string) (*ZenodoMetaResponse, error) {
	description := fmt.Sprintf("Lua module created using PIXLISE (https://pixlise.org). %v", module.DataModule.Comments)

	metadata := ZenodoDepositionMetadata{
		Title:       module.DataModule.Name,
		UploadType:  "software",
		Description: description,
		Creators: []struct {
			Name        string `json:"name"`
			Affiliation string `json:"affiliation"`
		}{
			{
				Name: module.DataModule.Origin.Creator.Name,
			},
		},
	}

	return addMetadataToDeposition(deposition, metadata, accessToken)
}

func publishDeposition(deposition ZenodoDepositionResponse, accessToken string) (*ZenodoPublishResponse, error) {
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

	zenodoResponse := ZenodoPublishResponse{}
	err = json.Unmarshal(publishBody, &zenodoResponse)
	if err != nil {
		return nil, err
	}

	return &zenodoResponse, nil
}

func PublishModuleToZenodo(module modules.DataModuleSpecificVersionWire) (*ZenodoPublishResponse, error) {
	zenodoResponse := ZenodoPublishResponse{}

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

	_, err = addModuleMetadataToDeposition(*deposition, module, accessToken)
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
