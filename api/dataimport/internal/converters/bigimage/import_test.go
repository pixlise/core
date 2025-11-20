package importBigImage

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_importBigImage_Import() {
	var im = BigImage{}
	out, id, err := im.Import("./test-data", "", "Multipager", &logger.StdOutLoggerForTest{})

	fmt.Printf("%v|%v|%v", len(out.PerPMCData), id, err)

	// Output:
	// 590531|./test-data|<nil>
}
