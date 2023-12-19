package coreg

type CoregJobRequest struct {
	JobID       string `json:"jobId"`
	Environment string `json:"environment"`
	TriggerUrl  string `json:"triggerUrl"`
}

type CoregJobResult struct {
	JobID       string `json:"jobId" bson:"_id"`
	Environment string `json:"environment"`

	MarsViewerExportUrl string `json:"marsViewerExportUrl"`

	ContextImageUrls []string `json:"contextImageUrls"`
	MappedImageUrls  []string `json:"mappedImageUrls"`
	WarpedImageUrls  []string `json:"warpedImageUrls"`
	BaseImageUrl     string   `json:"baseImageUrl"`
}
