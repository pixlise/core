# Image Pyramid Generator

Generate and serve multi-resolution pyramidal TIFFs for fast web-based image viewing.

## Overview

This package provides:
- **Pyramid Generation**: Convert large TIFFs → pyramidal TIFFs with multiple zoom levels
- **Tile Extraction**: Extract tiles on-demand from pyramidal TIFFs
- **Protobuf Metadata**: Generate ImagePyramid protobuf structures

## Usage

### 1. Generate Pyramidal TIFF

```go
import "github.com/pixlise/core/v4/api/imagepyramid"

// Generate pyramid from source TIFF
input := imagepyramid.ImageInput{
    Path:    "/path/to/large-image.tiff",
    Channel: 0,  // Extract first channel (or 0 for grayscale)
}

config := imagepyramid.GeneratorConfig{
    TileSize:    256,      // Tile dimensions
    Compression: "jpeg",   // "jpeg" or "deflate"
    Quality:     85,       // JPEG quality (1-100)
}

err := imagepyramid.GeneratePyramidalTIFF(
    input,
    "/output/pyramid.tiff",
    config,
)
```

### 2. Extract Tiles

```go
// Extract tile at zoom level 2, position (4, 3)
tileData, err := imagepyramid.ExtractTile(
    "/output/pyramid.tiff",
    2,    // zoom level (0 = most zoomed out)
    4,    // x coordinate
    3,    // y coordinate
    256,  // tile size
)

// tileData is JPEG-encoded image bytes
// Serve directly to HTTP response or save to file
```

### 3. Get Pyramid Metadata

```go
pyramid, err := imagepyramid.GetPyramidInfo("/output/pyramid.tiff")

fmt.Printf("Pyramid levels: %d\n", len(pyramid.Pyramid))
fmt.Printf("Base resolution: %.0fx%.0f\n",
    pyramid.Bounds.Max.X,
    pyramid.Bounds.Max.Y)

for i, layer := range pyramid.Pyramid {
    fmt.Printf("Level %d: %.0fx%.0f, %d tiles\n",
        i,
        layer.Bounds.Max.X,
        layer.Bounds.Max.Y,
        len(layer.Tiles))
}
```

## API Endpoint Example

```go
// In your API router
router.HandleFunc("/tiles/{scanId}/{zoom}/{x}/{y}.jpg", TileHandler)

func TileHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    scanId := vars["scanId"]
    zoom, _ := strconv.Atoi(vars["zoom"])
    x, _ := strconv.Atoi(vars["x"])
    y, _ := strconv.Atoi(vars["y"])

    // Get pyramid path for scan
    pyramidPath := fmt.Sprintf("/data/pyramids/%s.tiff", scanId)

    // Extract tile
    tileData, err := imagepyramid.ExtractTile(pyramidPath, zoom, x, y, 256)
    if err != nil {
        http.Error(w, err.Error(), 404)
        return
    }

    // Serve tile
    w.Header().Set("Content-Type", "image/jpeg")
    w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache 1 year
    w.Write(tileData)
}
```

## Frontend Integration

```javascript
// Use with Leaflet
const map = L.map('map', {
    crs: L.CRS.Simple,
    minZoom: 0,
    maxZoom: 4
});

L.tileLayer('/api/tiles/scan123/{z}/{x}_{y}.jpg', {
    tileSize: 256,
    noWrap: true
}).addTo(map);
```

## Testing

```bash
# Run tests
go test -v ./api/imagepyramid

# Test with your TIFF files
# Edit test file to point to your test images
```

## Performance

**Pyramid Generation:**
- 600 MB TIFF → ~400 MB pyramidal TIFF
- Time: ~10-30 seconds (depends on compression)

**Tile Extraction:**
- ~2-5ms per tile (from pyramidal TIFF)
- ~15ms total (including JPEG encoding)

## File Structure

```
Pyramidal TIFF:
├── Page 0 (Level 0) - 128×128   (thumbnail)
├── Page 1 (Level 1) - 256×256
├── Page 2 (Level 2) - 512×512
├── Page 3 (Level 3) - 1024×1024
└── Page 4 (Level 4) - 2048×2048 (full resolution)

Each page is internally tiled (256×256 tiles)
```

## Notes

- Requires libvips 8.17+ installed
- Polygons and point mapping not yet implemented (coming soon)
- For now, points/polygons fields in protobuf are set to 0
