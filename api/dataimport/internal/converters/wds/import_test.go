package importwds

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_importwds_Import() {
	var im = ImageMaps{}
	out, id, err := im.Import("./test-data", "", "2_Zagami5", &logger.StdOutLoggerForTest{})

	fmt.Printf("%v|%v|%v", len(out.PerPMCData), id, err)

	// Output:
	// 590531|./test-data|<nil>
}
