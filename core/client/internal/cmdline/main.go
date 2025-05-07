package main

import (
	"fmt"

	"github.com/pixlise/core/v4/core/client"
)

func main() {
	// Try to load the config file
	_, err := client.Authenticate()
	fmt.Printf("%v\n", err)

	// counts, err := client.GetSpectrum(sock, "500302337", 15, protos.SpectrumType_SPECTRUM_NORMAL, "A")
	// fmt.Printf("%v|%v\n", len(counts), err)
}
