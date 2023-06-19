package zenodoModels

// Structures we store
type DOIRelatedIdentifier struct {
	Identifier string `json:"identifier"`
	Relation   string `json:"relation"`
}

type DOICreator struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
	Orcid       string `json:"orcid"`
}

type DOIContributor struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation"`
	Orcid       string `json:"orcid"`
	Type        string `json:"type"`
}

type DOIMetadata struct {
	Title              string                 `json:"title"`
	Creators           []DOICreator           `json:"creators"`
	Description        string                 `json:"description"`
	Keywords           string                 `json:"keywords"`
	Notes              string                 `json:"notes"`
	RelatedIdentifiers []DOIRelatedIdentifier `json:"relatedIdentifiers"`
	Contributors       []DOIContributor       `json:"contributors"`
	References         string                 `json:"references"`
	Version            string                 `json:"version"`
	DOI                string                 `json:"doi"`
	DOIBadge           string                 `json:"doiBadge"`
	DOILink            string                 `json:"doiLink"`
}

// Structures received from Zenodo
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
