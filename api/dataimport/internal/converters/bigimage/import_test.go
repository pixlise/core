package importBigImage

import (
	"testing"

	"github.com/pixlise/core/v4/core/logger"
)

func TestImportBigImage(t *testing.T) {
	// Importing an image with different page dimensions should error
	var im = BigImage{}
	_, _, err := im.Import("./test-data", "", "DimensionsMismatch", &logger.StdOutLoggerForTest{})

	// Should fail with dimension mismatch error
	if err == nil {
		t.Fatal("Expected error for multi-page TIFF with different page dimensions, but got none")
	}

	t.Logf("Got expected error: %v", err)
}
