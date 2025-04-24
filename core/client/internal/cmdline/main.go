package main

import (
	"flag"
	"fmt"

	"github.com/pixlise/core/v4/core/client"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "configFile", "", "Path to config file")
	flag.Parse()

	// Try to load the config file
	sock, err := client.Authenticate(configFile)
	fmt.Printf("%v\n", err)

	counts, err := client.GetSpectrum(sock, "500302337", 15, protos.SpectrumType_SPECTRUM_NORMAL, "A")
	fmt.Printf("%v|%v\n", len(counts), err)
}
