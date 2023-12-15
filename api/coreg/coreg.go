package coreg

import protos "github.com/pixlise/core/v3/generated-protos"

func NewJobFromMVExport(jobId string, mve *protos.MarsViewerExport) *CoregJobRequest {
	var job CoregJobRequest
	for _, observation := range mve.Observations {
		job.ContextImageUrls = append(job.ContextImageUrls, observation.ContextImageUrl)
	}
	for _, overlay := range mve.WarpedOverlayImages {
		job.WarpedImageUrls = append(job.WarpedImageUrls, overlay.WarpedImageUrl)
		job.MappedImageUrls = append(job.MappedImageUrls, overlay.MappedImageUrl)
	}
	job.BaseImageUrl = mve.BaseImageUrl

	// e.g. 0661b573-e36c
	job.JobID = jobId

	return &job
}

type CoregJobRequest struct {
	ContextImageUrls []string
	MappedImageUrls  []string
	WarpedImageUrls  []string
	BaseImageUrl     string
	JobID            string
	Environment      string
}

type CoregJobResult struct {
	ContextImageUrls []string `json:"contextImageUrls"`
	MappedImageUrls  []string `json:"mappedImageUrls"`
	WarpedImageUrls  []string `json:"warpedImageUrls"`
	BaseImageUrl     string   `json:"baseImageUrl"`
	JobID            string   `json:"jobId" bson:"_id"`
	Environment      string   `json:"environment"`
}
