package jobmanager

import (
	"github.com/pixlise/core/v4/api/services"
)

type JobManager struct {
	svcs *services.APIServices
}

func Create(svcs *services.APIServices) (*JobManager, error) {
	return &JobManager{svcs: svcs}, nil
}
