package bruker

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_bruker_Import() {
	b := Bruker{
		Downsample: 6,
	}
	l := logger.StdOutLoggerForTest{}
	_ /*out*/, impPath, err := b.Import("/home/peter/Documents/TornadoGSQ", "", "", &l)
	fmt.Printf("%v|%v\n", impPath, err)

	// Output:
	// <nil>|<nil>
}
