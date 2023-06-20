package endpoints

import (
	"net/http"

	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

var APIVersion = "4.0.0"

func getVersion() *protos.VersionResponse {
	result := &protos.VersionResponse{}
	result.Versions = []*protos.VersionResponse_Version{
		{
			Component: "API",
			Version:   APIVersion,
		},
	}
	return result
}

func GetVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result := getVersion()
	utils.SendProtoBinary(w, result)
}

func GetVersionJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result := getVersion()
	utils.SendProtoJSON(w, result)
}
