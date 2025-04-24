package client

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	protos "github.com/pixlise/core/v4/generated-protos"
)

type ClientConfig struct {
	Host     string
	User     string
	Pass     string
	ClientId string
	Domain   string
	Audience string
}

var configEnvVar = "PIXLISE_CLIENT_CONFIG"

// Authenticates using one of several methods:
// - If configPath is not empty, it will try load the config file, which must deserialise to a ClientConfig structure as above
// - If the config path is empty, it will try to load the config from an environment variable: PIXLISE_CLIENT_CONFIG
func Authenticate(configPath string) (*SocketConn, error) {
	cfg := ClientConfig{}
	var err error

	if len(configPath) > 0 {
		cfgBytes, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config %v. Error: %v", configPath, err)
		}

		err = json.Unmarshal(cfgBytes, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse custom config: %v", err)
		}
	} else {
		cfgStr := os.Getenv(configEnvVar)

		if len(cfgStr) <= 0 {
			return nil, fmt.Errorf("no config path and no environment variable (%v) defined. Cannot authenticate", configEnvVar)
		}

		err = json.Unmarshal([]byte(cfgStr), &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse custom config from environment variable: %v. Error: %v", configEnvVar, err)
		}
	}

	connectParams := ConnectInfo{
		Host: cfg.Host,
		User: cfg.User,
		Pass: cfg.Pass,
	}

	auth0Params := Auth0Info{
		ClientId: cfg.ClientId,
		Domain:   cfg.Domain,
		Audience: cfg.Audience,
	}

	return AuthenticateWithAuth0Info(connectParams, auth0Params)
}

func AuthenticateWithAuth0Info(connectParams ConnectInfo, auth0Params Auth0Info) (*SocketConn, error) {
	socket := &SocketConn{}
	err := socket.Connect(connectParams, auth0Params)
	return socket, err
}

func GetSpectrum(socket *SocketConn, scanId string, pmc int, spectrumType protos.SpectrumType, detector string) ([]int32, error) {
	req := &protos.SpectrumReq{ScanId: scanId, Entries: &protos.ScanEntryRange{Indexes: []int32{int32(pmc)}}}
	msg := &protos.WSMessage{Contents: &protos.WSMessage_SpectrumReq{
		SpectrumReq: req,
	}}

	if err := socket.SendMessage(msg); err != nil {
		return []int32{}, err
	}

	resps := socket.WaitForMessages(1, time.Duration(5)*time.Second)
	if len(resps) != 1 {
		return []int32{}, fmt.Errorf("Expected 1 response, got %v", len(resps))
	}

	resp := resps[0].GetSpectrumResp()
	for _, spectra := range resp.SpectraPerLocation {
		for _, spectrum := range spectra.Spectra {
			if spectrum.Type == spectrumType && spectrum.Detector == detector {
				spectrumCounts := zeroRunDecode(spectrum.Counts)
				return spectrumCounts, nil
			}
		}
	}

	return []int32{}, nil
}
