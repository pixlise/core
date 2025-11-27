package pyramid

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// TestImportBigTIFF tests the main ImportBigTIFF function
func TestImportBigTIFF(t *testing.T) {
	// Use the existing test data
	inputTiff := "./test-data/Big_Import/pyramid/Multi_page24bpp.tif"

	if _, err := os.Stat(inputTiff); os.IsNotExist(err) {
		t.Skipf("Test image not found: %s", inputTiff)
	}

	// Create output directory structure that matches what the pipeline expects
	tmpDir, err := os.MkdirTemp("", "pyramid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create scan directory (simulating the BigTiff scan ID)
	scanDir := filepath.Join(tmpDir, "TestScan")
	if err := os.MkdirAll(scanDir, 0755); err != nil {
		t.Fatalf("Failed to create scan dir: %v", err)
	}

	testCases := []struct {
		name        string
		pageNum     int
		tileSize    int
		tileQuality int
		expectPNG   bool
	}{
		{"Page 0 - Default JPEG", 0, 254, 85, false},
		{"Page 0 - PNG Quality", 0, 254, 100, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Construct paths as the pipeline would
			// fromImgFile would be: /path/pyramid/PY_Multi_page24bpp.tif
			// outImgFile would be: /path/TestScan/PY_Multi_page24bpp.png
			fromImgFile := filepath.Join(filepath.Dir(inputTiff), "PY_Multi_page24bpp.tif")
			outImgFile := filepath.Join(scanDir, "PY_Multi_page24bpp.png")

			// Call the main function we're testing
			pyramid, err := ImportBigTIFF(fromImgFile, outImgFile, tc.pageNum, tc.tileSize, tc.tileQuality, &logger.StdOutLoggerForTest{})
			if err != nil {
				t.Fatalf("ImportBigTIFF failed: %v", err)
			}

			// Verify pyramid structure
			if pyramid == nil {
				t.Fatal("Expected non-nil pyramid")
			}

			// Verify tile size
			if pyramid.TileSize != uint32(tc.tileSize) {
				t.Errorf("Expected TileSize %d, got %d", tc.tileSize, pyramid.TileSize)
			}

			// Verify bounds are set
			if pyramid.Bounds == nil {
				t.Error("Expected non-nil Bounds")
			}

			// Verify layers exist
			if len(pyramid.Pyramid) == 0 {
				t.Error("Expected at least one pyramid layer")
			}

			// Verify image prefixes
			if len(pyramid.ImagePrefixes) != 1 {
				t.Errorf("Expected 1 image prefix, got %d", len(pyramid.ImagePrefixes))
			}

			t.Logf("Pyramid generated successfully:")
			t.Logf("  TileSize: %d", pyramid.TileSize)
			t.Logf("  Layers: %d", len(pyramid.Pyramid))
			t.Logf("  Bounds: [%.0f,%.0f] to [%.0f,%.0f]",
				pyramid.Bounds.Min.X, pyramid.Bounds.Min.Y,
				pyramid.Bounds.Max.X, pyramid.Bounds.Max.Y)
			t.Logf("  ImagePrefix: %s", pyramid.ImagePrefixes[0])
		})
	}
}

// TestParseDZIFile tests the DZI XML parsing function
func TestParseDZIFile(t *testing.T) {
	// Create a temporary DZI file
	tmpDir, err := os.MkdirTemp("", "dzi-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dziContent := `<?xml version="1.0" encoding="UTF-8"?>
					<Image xmlns="http://schemas.microsoft.com/deepzoom/2008"
						Format="jpg"
						Overlap="0"
						TileSize="254">
					<Size Width="1024" Height="768"/>
					</Image>`

	dziPath := filepath.Join(tmpDir, "test.dzi")
	if err := os.WriteFile(dziPath, []byte(dziContent), 0644); err != nil {
		t.Fatalf("Failed to write DZI file: %v", err)
	}

	// Test parsing
	dzi, err := parseDZIFile(dziPath)
	if err != nil {
		t.Fatalf("parseDZIFile failed: %v", err)
	}

	// Verify parsed values
	if dzi.Format != "jpg" {
		t.Errorf("Expected Format 'jpg', got '%s'", dzi.Format)
	}
	if dzi.Overlap != 0 {
		t.Errorf("Expected Overlap 0, got %d", dzi.Overlap)
	}
	if dzi.TileSize != 254 {
		t.Errorf("Expected TileSize 254, got %d", dzi.TileSize)
	}
	if dzi.Size.Width != 1024 {
		t.Errorf("Expected Width 1024, got %d", dzi.Size.Width)
	}
	if dzi.Size.Height != 768 {
		t.Errorf("Expected Height 768, got %d", dzi.Size.Height)
	}
}

// TestBuildImagePyramidProto tests the proto construction logic
func TestBuildImagePyramidProto(t *testing.T) {
	// Create test DZI data
	dzi := &dziImage{
		Format:   "jpg",
		Overlap:  0,
		TileSize: 16,
		Size: dziSize{
			Width:  32,
			Height: 32,
		},
	}

	pyramid := buildImagePyramidProto(dzi, "TestScan", "TestImage", &logger.StdOutLoggerForTest{})

	// Verify pyramid structure
	if pyramid == nil {
		t.Fatal("Expected non-nil pyramid")
	}

	// Verify tile size
	if pyramid.TileSize != 16 {
		t.Errorf("Expected TileSize 16, got %d", pyramid.TileSize)
	}

	// Verify bounds match image dimensions
	if pyramid.Bounds.Max.X != 32 {
		t.Errorf("Expected max X bound 32, got %.0f", pyramid.Bounds.Max.X)
	}
	if pyramid.Bounds.Max.Y != 32 {
		t.Errorf("Expected max Y bound 32, got %.0f", pyramid.Bounds.Max.Y)
	}

	// Verify layers were created
	if len(pyramid.Pyramid) != 2 {
		t.Error("Expected 2 pyramid layers, got", len(pyramid.Pyramid))
	}
	fmt.Printf("%+v\n", pyramid.Pyramid)

	// Verify each layer has tiles
	for i, layer := range pyramid.Pyramid {
		if len(layer.Tiles) == 0 {
			t.Errorf("Layer %d has no tiles", i)
		}
		if layer.Bounds == nil {
			t.Errorf("Layer %d has nil bounds", i)
		}
	}
	if countTotalTiles((pyramid)) != 5 {
		t.Errorf("Expected total of 5 tiles, got %d", countTotalTiles(pyramid))
	}

	// Verify image prefix
	expectedPrefix := "TestScan/TestImage"
	if len(pyramid.ImagePrefixes) != 1 || pyramid.ImagePrefixes[0] != expectedPrefix {
		t.Errorf("Expected ImagePrefix '%s', got %v", expectedPrefix, pyramid.ImagePrefixes)
	}

	t.Logf("Pyramid proto created successfully:")
	t.Logf("  Layers: %d", len(pyramid.Pyramid))
	t.Logf("  Total tiles: %d", countTotalTiles(pyramid))
}

// Helper function to count total tiles across all layers
func countTotalTiles(pyramid *protos.ImagePyramid) int {
	total := 0
	for _, layer := range pyramid.Pyramid {
		total += len(layer.Tiles)
	}
	return total
}

// TestVerifyPyramids_Success tests that identical pyramids pass verification
func TestVerifyPyramids_Success(t *testing.T) {
	pyramid1 := createTestPyramid(1024, 768, 254)
	pyramid2 := createTestPyramid(1024, 768, 254)

	err := VerifyPyramids(pyramid1, pyramid2, 0, 1)
	if err != nil {
		t.Errorf("Expected no error for matching pyramids, got: %v", err)
	}
}

// TestVerifyPyramids_DimensionMismatch tests that pyramids with different dimensions fail
func TestVerifyPyramids_DimensionMismatch(t *testing.T) {
	testCases := []struct {
		name    string
		width1  int
		height1 int
		width2  int
		height2 int
	}{
		{"Different width", 1024, 768, 2048, 768},
		{"Different height", 1024, 768, 1024, 1536},
		{"Both different", 1024, 768, 2048, 1536},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pyramid1 := createTestPyramid(tc.width1, tc.height1, 254)
			pyramid2 := createTestPyramid(tc.width2, tc.height2, 254)

			err := VerifyPyramids(pyramid1, pyramid2, 0, 1)
			if err == nil {
				t.Error("Expected error for dimension mismatch but got none")
			}
			t.Logf("Got expected error: %v", err)
		})
	}
}

// TestVerifyPyramids_TileSizeMismatch tests that pyramids with different tile sizes fail
func TestVerifyPyramids_TileSizeMismatch(t *testing.T) {
	pyramid1 := createTestPyramid(1024, 768, 254)
	pyramid2 := createTestPyramid(1024, 768, 512) // Different tile size

	err := VerifyPyramids(pyramid1, pyramid2, 0, 1)
	if err == nil { // Fail test if there's no error
		t.Error("Expected error for tile size mismatch but got none")
	}
	t.Logf("Got expected error: %v", err)
}

// Helper function to create a test pyramid with specific dimensions
func createTestPyramid(width, height, tileSize int) *protos.ImagePyramid {
	dzi := &dziImage{
		Format:   "jpg",
		Overlap:  0,
		TileSize: tileSize,
		Size: dziSize{
			Width:  width,
			Height: height,
		},
	}

	return buildImagePyramidProto(dzi, "TestScan", "TestImage", &logger.StdOutLoggerForTest{})
}
