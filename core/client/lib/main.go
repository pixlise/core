package main

import (
	"C"

	"fmt"
)
import (
	"errors"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
)

var socket *client.SocketConn

//export authenticate
func authenticate(configFile string) {
	fmt.Printf("configFile: \"%v\"\n", configFile)
	var err error

	// Try to load the config file
	socket, err = client.Authenticate(configFile)
	fmt.Printf("%v\n", err)

	fmt.Printf("Authenticate()\n")
}

//export getSpectrum
func getSpectrum(scanId string, pmc int, spectrumType int, detector string) ([]int32, error) {
	if socket == nil {
		return []int32{}, errors.New("Not authenticated")
	}

	return client.GetSpectrum(socket, scanId, pmc, protos.SpectrumType(spectrumType), detector)
}

func main() {
}
