package servicesMock

import (
	"net/http"

	"github.com/pixlise/core/v4/core/jwtparser"
)

type MockJWTReader struct {
	InfoToReturn *jwtparser.JWTUserInfo
}

func (m MockJWTReader) GetUserInfo(*http.Request) (jwtparser.JWTUserInfo, error) {
	if m.InfoToReturn != nil {
		return *m.InfoToReturn, nil
	}
	//This user id is real don't change it....
	return jwtparser.JWTUserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Email:  "niko@spicule.co.uk",
		Permissions: map[string]bool{
			"read:data-analysis": true,
		},
	}, nil
}

func (m MockJWTReader) GetValidator() jwtparser.JWTInterface {
	return nil
}
