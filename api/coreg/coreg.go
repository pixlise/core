package coreg

import protos "github.com/pixlise/core/v3/generated-protos"

type CoregJobRequest struct {
	JobID       string `json:"jobId"`
	Environment string `json:"environment"`
	TriggerUrl  string `json:"triggerUrl"`
}

type CoregJobResult struct {
	JobID       string `json:"jobId" bson:"_id"`
	Environment string `json:"environment"`

	MarsViewerExportSource *protos.MarsViewerExport `json:"marsViewerExportSource"`

	ContextImageUrls []string `json:"contextImageUrls"`
	MappedImageUrls  []string `json:"mappedImageUrls"`
	WarpedImageUrls  []string `json:"warpedImageUrls"`
	BaseImageUrl     string   `json:"baseImageUrl"`
}
