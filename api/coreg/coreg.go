package coreg

type CoregJobRequest struct {
	JobID       string `json:"jobId"`
	Environment string `json:"environment"`
	TriggerUrl  string `json:"triggerUrl"`
}

type CoregFile struct {
	OriginalUri string
	NewUri      string
	Completed   bool
}

type CoregJobResult struct {
	JobID       string `json:"jobId" bson:"_id"`
	Environment string `json:"environment"`

	MarsViewerExportUrl string `json:"marsViewerExportUrl"`

	ContextImageUrls []CoregFile `json:"contextImageUrls"`
	MappedImageUrls  []CoregFile `json:"mappedImageUrls"`
	WarpedImageUrls  []CoregFile `json:"warpedImageUrls"`
	BaseImageUrl     CoregFile   `json:"baseImageUrl"`
	AllCompleted     bool        `json:"allCompleted"`
}
