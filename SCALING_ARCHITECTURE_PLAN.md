# PIXLISE Scaling Architecture Plan
## Handling Large-Scale Geological Datasets

**Context:** PIXLISE currently handles rover-scale datasets (~30k points, 752×580 images). We need to scale to terrestrial geology datasets with:
- **Images:** 30k × 20k pixels, 2GB+ TIFF files
- **Point Data:** Potentially millions of XRF measurement points
- **Use Case:** Zoom in/out workflows, viewport-based exploration

---

## Table of Contents
1. [Current Architecture Analysis](#current-architecture-analysis)
2. [Scaling Challenges](#scaling-challenges)
3. [Proposed Architecture](#proposed-architecture)
4. [Implementation Plan](#implementation-plan)
5. [Migration Strategy](#migration-strategy)
6. [Performance Projections](#performance-projections)

---

## Current Architecture Analysis

### Images: Single-File Streaming Model

**Current Flow:**
```
HTTP GET /images/{scan}/{filename}
  ↓
MongoDB lookup (metadata)
  ↓
S3 retrieval (full file)
  ↓
Optional: Scale + mark locations (in-memory)
  ↓
Stream entire file to client
```

**Key Components:**
- **Storage:** S3 with `Images/` (originals) + `Image-Cache/` (transformed)
- **Formats:** PNG, JPG, TIF (RGBU 32-bit float for multi-channel)
- **Caching:**
  - ETag/If-Modified-Since HTTP caching
  - Scale steps of 200px to limit cache variants
  - Beam location overlays cached separately
- **Location:** `api/endpoints/Image.go:76-257`

**Current Optimizations:**
- Aspect-preserving downscaling (ApproxBiLinear)
- Cache-Control headers for CDN compatibility
- 200px stepping to limit cache explosion

**Limitations:**
- ❌ Full image loaded into memory for any transformation
- ❌ Single HTTP response (no chunking)
- ❌ Client receives entire file even if viewing small region
- ❌ No streaming/progressive loading

---

### Point Data: Full-Dataset Loading Model

**Current Flow:**
```
Request for scan data/quantification
  ↓
S3 retrieval: Scans/{scanId}/dataset.bin
  ↓
Unmarshal entire protobuf (~500 MB)
  ↓
Cache in /tmp (5 min TTL, 200 MB limit)
  ↓
Linear search/iteration for queries
```

**Data Structure:**
```protobuf
Experiment {
    repeated Location locations          # ALL points in single array
}

Location {
    string id                            # PMC (point number)
    BeamLocation beam                    # x,y,z + image_i, image_j
    repeated DetectorSpectrum detectors  # XRF spectra (4096 channels)
    repeated MetaDataItem meta
}

DetectorSpectrum {
    repeated int32 spectrum              # ~16 KB per spectrum
}
```

**Storage Details:**
- **File Format:** Binary protobuf (`dataset.bin`)
- **Location:** `Scans/<scanId>/dataset.bin` in S3
- **Typical Size:** 10-500 MB per scan
- **Compression:** Spectra use RLE/zero-run encoding

**Query Patterns:**
| Operation | Method | Complexity |
|-----------|--------|------------|
| Get all points | Load full Experiment, iterate | O(n) |
| Get point by index | Array access after load | O(1) after load |
| Get point by PMC | Linear search on `Location.id` | O(n) |
| ROI points | Decode compressed indices, map to PMCs | O(roi_size) after load |
| Spatial query | Linear scan all locations checking `image_i`, `image_j` | O(n) |

**Current Optimizations:**
- File cache in `/tmp` (5-minute TTL)
- ROI indices use run-length encoding
- Spectrum compression (RLE, zero-run)

**Limitations:**
- ❌ Entire dataset loads into memory (no streaming)
- ❌ No spatial indexing (can't query "points in viewport")
- ❌ Linear search required for spatial queries
- ❌ Quantifications stored separately (double file load)
- ❌ Cache eviction at 200 MB total (tiny for large datasets)

---

## Scaling Challenges

### Image Problems at 30k × 20k (2GB TIFF)

| Challenge | Impact |
|-----------|--------|
| **Memory footprint** | 2.4 GB uncompressed (30k × 20k × 4 bytes RGBA) |
| **Network transfer** | Even compressed PNG ~500 MB - unusable for full transfer |
| **Transformation time** | Downscaling requires loading entire source image |
| **Cache explosion** | With beam locations + scale variants, cache grows massively |
| **Client rendering** | Browser canvas limits, memory pressure |

**Example:**
- User zooms to 10% of image → still downloads 100% of data
- Marking beam locations → regenerates entire 2 GB image

---

### Point Data Problems at 1M+ Points

| Challenge | Impact |
|-----------|--------|
| **File size** | 1M points × 4 detectors × 16 KB spectrum = **64 GB raw** |
| **Load time** | Even with compression, loading 10+ GB file takes minutes |
| **Memory usage** | Entire dataset must fit in API server RAM |
| **Spatial queries** | Finding points in viewport requires scanning all 1M entries |
| **Quantification coupling** | Separate file doubles I/O for every analysis workflow |

**Example:**
- User views small region → loads entire 10 GB dataset
- Querying "points in image region [1000:2000, 500:1500]" → scans all 1M points

---

### Combined Workflow Problem

**Typical User Flow:**
1. Load image → **2 GB transfer**
2. Zoom to region of interest → **still using 2 GB**
3. Query points in viewport → **load 10 GB dataset, scan linearly**
4. Run quantification → **load separate 5 GB quant file**

**Total:** 17 GB transferred and loaded for viewing 1% of the data

---

## Proposed Architecture

### Philosophy: Spatial Indexing Everywhere

Both images and point data need **tile-based / spatial partitioning** with:
- ✅ Load only what's visible
- ✅ Progressive detail (zoom in = more data)
- ✅ Spatial indexing for fast viewport queries
- ✅ CDN-friendly (immutable tiles)

---

## Part 1: Tile-Based Image Architecture

### Tile Pyramid Structure

Generate multiple zoom levels during dataset import:

```
Original: 30,000 × 20,000 pixels
├─ Level 0 (100%): 30,000 × 20,000 → 118 × 79 tiles @ 256×256
├─ Level 1 (50%):  15,000 × 10,000 → 59 × 40 tiles
├─ Level 2 (25%):   7,500 ×  5,000 → 30 × 20 tiles
├─ Level 3 (12.5%): 3,750 ×  2,500 → 15 × 10 tiles
└─ Level 4 (6.25%): 1,875 ×  1,250 → 8 × 5 tiles (overview)
```

**Tile Naming Convention:**
- Slippy map standard: `z{zoom}/x{col}_y{row}.png`
- Zoom 0 = highest resolution, higher zoom = lower resolution
- (Or invert: zoom 0 = overview, matches common web map conventions)

**Storage Structure:**
```
Images/
  {scan}/
    {image-id}/
      metadata.json              # Dimensions, zoom levels, tile size, format
      tiles/
        z0/                      # Full resolution
          x0_y0.png
          x0_y1.png
          x1_y0.png
          ...
        z1/                      # 50% scale
          x0_y0.png
          ...
      tiles-with-locations/      # Pre-rendered with beam overlays
        z0/
          x0_y0.png
          ...
```

**Metadata JSON:**
```json
{
  "width": 30000,
  "height": 20000,
  "tileSize": 256,
  "format": "png",
  "zoomLevels": 5,
  "bounds": {
    "minZoom": 0,
    "maxZoom": 4
  },
  "hasLocationOverlay": true,
  "generatedAt": "2025-11-11T12:00:00Z"
}
```

---

### New REST Endpoints

**Tile Metadata:**
```
GET /image-tiles/{scan}/{image-id}/metadata
```
Response:
```json
{
  "width": 30000,
  "height": 20000,
  "tileSize": 256,
  "zoomLevels": 5
}
```

**Individual Tile:**
```
GET /image-tiles/{scan}/{image-id}/{z}/{x}/{y}
Query params:
  ?with-locations=true  → serve from tiles-with-locations/
```
Response: 256×256 PNG binary with aggressive caching headers

**Implementation Location:** `api/endpoints/ImageTile.go`

---

### Tile Generation Pipeline

**Import-Time Generation:**

```go
// In api/dataimport/internal/output/imageTiler.go

func TileLargeImage(sourcePath string, scanId string, imageId string, s3 FileAccess) error {
    // 1. Read TIFF metadata (avoid loading full image)
    width, height, err := ReadTIFFDimensions(sourcePath)
    if err != nil {
        return err
    }

    // 2. Determine if tiling needed
    if width < 4096 && height < 4096 {
        return nil  // Use legacy single-file approach
    }

    // 3. Calculate zoom levels (each level is 50% of previous)
    maxZoom := calculateZoomLevels(width, height, 256)

    // 4. Generate tiles for each zoom level
    for zoom := 0; zoom < maxZoom; zoom++ {
        scale := math.Pow(0.5, float64(zoom))
        scaledWidth := int(float64(width) * scale)
        scaledHeight := int(float64(height) * scale)

        err := generateTilesForLevel(
            sourcePath,
            scanId, imageId, zoom,
            scaledWidth, scaledHeight,
            256, s3,
        )
        if err != nil {
            return err
        }
    }

    // 5. Generate metadata.json
    metadata := ImageTileMetadata{
        Width:      width,
        Height:     height,
        TileSize:   256,
        ZoomLevels: maxZoom,
        Format:     "png",
    }

    return saveMetadata(s3, scanId, imageId, metadata)
}

func generateTilesForLevel(
    sourcePath string,
    scanId, imageId string, zoom int,
    width, height, tileSize int,
    s3 FileAccess,
) error {
    // Use streaming TIFF decoder
    decoder := tiff.NewDecoder(sourcePath)

    // Calculate tile grid
    tilesX := (width + tileSize - 1) / tileSize
    tilesY := (height + tileSize - 1) / tileSize

    for ty := 0; ty < tilesY; ty++ {
        for tx := 0; tx < tilesX; tx++ {
            // Extract tile region from source (with streaming)
            tileImg := extractTileRegion(decoder, tx, ty, tileSize, zoom)

            // Encode as PNG
            var buf bytes.Buffer
            png.Encode(&buf, tileImg)

            // Upload to S3
            path := fmt.Sprintf("Images/%s/%s/tiles/z%d/x%d_y%d.png",
                scanId, imageId, zoom, tx, ty)

            s3.WriteObject(path, buf.Bytes())
        }
    }

    return nil
}
```

**Streaming TIFF Reading:**
- Use `github.com/google/tiff` or `libtiff` bindings
- Read only required tile regions, not entire image
- Decode on-the-fly during tile generation

---

### Beam Location Overlay Strategy

**Option A: Pre-Generate Location Tiles (RECOMMENDED)**

During import, after generating base tiles:
```go
func GenerateLocationOverlayTiles(
    scanId, imageId string,
    beamLocations []BeamLocation,
    metadata ImageTileMetadata,
    s3 FileAccess,
) error {
    // For each zoom level
    for zoom := 0; zoom < metadata.ZoomLevels; zoom++ {
        scale := math.Pow(0.5, float64(zoom))

        // Spatial index beam locations to tiles
        tileBeams := spatiallyIndexBeamsToTiles(beamLocations, scale, 256)

        // For each tile
        for tileKey, beams := range tileBeams {
            // Load base tile
            baseTile := loadTile(s3, scanId, imageId, zoom, tileKey.x, tileKey.y)

            // Draw beam locations on tile
            overlayTile := drawBeamMarkers(baseTile, beams, scale)

            // Save to tiles-with-locations/
            saveTile(s3, scanId, imageId, zoom, tileKey.x, tileKey.y, overlayTile, true)
        }
    }
}
```

**Pros:**
- ✅ Fast tile serving (pre-rendered)
- ✅ No runtime computation
- ✅ CDN-friendly (immutable)

**Cons:**
- ❌ Doubles storage (base + overlay tiles)
- ❌ Must regenerate if beam locations updated

**Storage Impact:**
- 30k×20k image → ~9,300 tiles across all zoom levels
- ~50 KB avg per tile → **~465 MB total**
- With overlays: **~930 MB** (still much better than 2 GB single file)

**Option B: Dynamic Overlay (Not Recommended)**
- Generate overlay on tile request
- Query beam locations in tile bounds from spatial index
- Draw markers on base tile
- Pro: Less storage, Con: Slower, more CPU

**Recommendation:** Use Option A. Storage is cheap, latency matters.

---

### Caching Strategy

**CloudFront CDN Configuration:**
```
Behavior: /image-tiles/*
  Cache Policy:
    - Cache-Control: max-age=31536000 (1 year)
    - ETag: S3 object hash
  Origin: S3 DatasetsBucket
  Compress: true
  Viewer Protocol: HTTPS only
```

**Why Tiles Cache Better:**
- Immutable once generated (content-addressed by z/x/y)
- Small size (256×256 PNG ~50 KB) → fast edge cache
- High hit rate (users explore similar regions)

**Backend Caching:**
- Remove in-memory image transformations (no longer needed)
- Let CDN handle all caching
- S3 bucket versioning for invalidation on re-import

---

### Backward Compatibility

**Legacy Endpoint Behavior:**
```go
// api/endpoints/Image.go

func GetImage(params apiRouter.ApiHandlerStreamParams) (*s3.GetObjectOutput, ...) {
    scanId := params.PathParams["scan"]
    filename := params.PathParams["filename"]

    // Check if tiled version exists
    metadataPath := fmt.Sprintf("Images/%s/%s/metadata.json", scanId, filename)
    if params.Svcs.FS.ObjectExists(metadataPath, params.Svcs.Config.DatasetsBucket) {
        // Redirect to tile-based viewer
        // OR serve lowest zoom level as preview thumbnail
        return serveLowestZoomPreview(params, scanId, filename)
    }

    // Fall back to current single-file behavior for small images
    return serveFullImage(params, scanId, filename)
}
```

**Migration Path:**
- Small images (< 4096×4096) stay as single files
- Large images automatically tiled on import
- Existing datasets gradually migrated (see Migration Strategy)

---

## Part 2: Spatial Point Data Architecture

### The Problem

Current point storage is **linear array** in a single protobuf file. We need:
- ✅ Query points by viewport (image region)
- ✅ Stream only visible data
- ✅ Spatial indexing for fast queries
- ✅ Integrate quantification data with points

---

### Proposed Solution: Spatial Tiles for Point Data

**Core Idea:** Partition point data into tiles matching the image tile grid.

**Why This Works:**
- Points have `image_i`, `image_j` coordinates
- Map points to same z/x/y tile system as images
- Query: "show points in viewport" → request specific tiles
- Perfect correspondence: image tile + point tile = complete view

---

### Point Data Tile Structure

**Storage:**
```
PointData/
  {scan}/
    metadata.json              # Point count, bounds, zoom levels
    tiles/
      z0/                      # Full resolution (all points)
        x0_y0.bin              # Points in this tile
        x0_y1.bin
        ...
      z1/                      # Downsampled (every 4th point)
        x0_y0.bin
        ...
      z4/                      # Heavily downsampled (overview)
        x0_y0.bin
```

**Tile Content Format:**
```protobuf
// New message type in data-formats/

message PointDataTile {
    repeated LocationSummary locations;
}

message LocationSummary {
    int32 pmc;                          // Point ID
    float x, y, z;                      // 3D coordinates
    float image_i, image_j;             // Image pixel coords

    // Spectral data reference (not inline)
    string spectra_tile_id;             // e.g., "spectra/z0/x5_y3.bin"
    int32 spectra_offset;               // Byte offset in tile file

    // Quantification data (if available)
    map<string, float> quant_values;    // e.g., {"Fe": 0.23, "Ca": 0.15}
}

// Separate storage for spectra (heavy data)
message SpectraTile {
    repeated SpectrumData spectra;
}

message SpectrumData {
    int32 pmc;
    repeated DetectorSpectrum detectors;  // Reuse existing proto
}
```

**Key Design Decisions:**

1. **Separate Location + Spectra Storage**
   - Locations (x,y,z, image coords): ~40 bytes per point
   - Spectra (4096 channels): ~16 KB per detector
   - Separation allows loading coordinates without heavy spectrum data

2. **Quantification Data Co-Located**
   - Embed quantification results in LocationSummary
   - Eliminates separate quantification file load
   - Trade-off: Tiles update when quantification runs

3. **Tile-Level Downsampling**
   - Zoom level 0: All points
   - Zoom level 1: Every 4th point (spatial subsampling)
   - Zoom level 2: Every 16th point
   - Provides LOD (level of detail) for overview mode

---

### Point-to-Tile Mapping

```go
// api/pointdata/tiler.go

func GeneratePointDataTiles(
    experiment *protos.Experiment,
    scanId string,
    imageTileMetadata ImageTileMetadata,
    s3 FileAccess,
) error {
    // 1. Build spatial index of all locations
    pointIndex := buildSpatialIndex(experiment.Locations, imageTileMetadata)

    // 2. For each zoom level
    for zoom := 0; zoom < imageTileMetadata.ZoomLevels; zoom++ {
        scale := math.Pow(0.5, float64(zoom))

        // 3. Determine downsampling factor
        sampleRate := calculateSampleRate(zoom)

        // 4. Assign points to tiles
        tileMap := assignPointsToTiles(pointIndex, zoom, scale, sampleRate)

        // 5. Write tile files
        for tileKey, points := range tileMap {
            locationTile := buildLocationTile(points)
            spectraTile := buildSpectraTile(points, experiment)

            // Save location tile
            saveTile(s3, scanId, zoom, tileKey, locationTile, "locations")

            // Save spectra tile (separate)
            saveTile(s3, scanId, zoom, tileKey, spectraTile, "spectra")
        }
    }

    // 6. Generate metadata
    metadata := PointDataMetadata{
        TotalPoints: len(experiment.Locations),
        ZoomLevels:  imageTileMetadata.ZoomLevels,
        TileSize:    256,  // Matches image tiles
    }

    return saveMetadata(s3, scanId, metadata)
}

func assignPointsToTiles(
    locations []*protos.Location,
    zoom int,
    scale float64,
    sampleRate int,
) map[TileKey][]*protos.Location {
    tileMap := make(map[TileKey][]*protos.Location)

    for idx, loc := range locations {
        // Downsample for lower zoom levels
        if idx % sampleRate != 0 {
            continue
        }

        // Calculate tile coordinates from image_i, image_j
        scaledI := loc.Beam.ImageI * float32(scale)
        scaledJ := loc.Beam.ImageJ * float32(scale)

        tileX := int(scaledI) / 256
        tileY := int(scaledJ) / 256

        key := TileKey{zoom: zoom, x: tileX, y: tileY}
        tileMap[key] = append(tileMap[key], loc)
    }

    return tileMap
}
```

**Downsampling Strategy:**
| Zoom Level | Scale | Sample Rate | Points (if 1M total) |
|------------|-------|-------------|----------------------|
| 0 | 100% | 1 (all) | 1,000,000 |
| 1 | 50% | 4 | 250,000 |
| 2 | 25% | 16 | 62,500 |
| 3 | 12.5% | 64 | 15,625 |
| 4 | 6.25% | 256 | 3,906 |

**Benefits:**
- Overview mode (zoom 4): Load only ~4k points
- Full detail (zoom 0): Load only visible tiles
- Progressive loading: zoom in → fetch higher detail

---

### New REST/WebSocket Endpoints

**Point Tile Metadata:**
```
GET /point-tiles/{scan}/metadata
```
Response:
```json
{
  "totalPoints": 1000000,
  "zoomLevels": 5,
  "tileSize": 256,
  "bounds": {
    "minX": 0, "maxX": 30000,
    "minY": 0, "maxY": 20000
  }
}
```

**Point Location Tile:**
```
GET /point-tiles/{scan}/{z}/{x}/{y}/locations
```
Response: Binary protobuf `PointDataTile` (~10 KB per tile)

**Spectra Tile:**
```
GET /point-tiles/{scan}/{z}/{x}/{y}/spectra
```
Response: Binary protobuf `SpectraTile` (~1-5 MB per tile depending on point density)

**WebSocket Batch Query:**
```protobuf
message PointTileBatchReq {
    string scanId;
    repeated TileCoordinate tiles;
    bool includeSpectra;
}

message TileCoordinate {
    int32 z, x, y;
}

message PointTileBatchResp {
    repeated TileData tiles;
}

message TileData {
    int32 z, x, y;
    PointDataTile locations;
    optional SpectraTile spectra;  // Only if requested
}
```

**Implementation:** `api/endpoints/PointTile.go` + `api/ws/handlers/pointTile.go`

---

### Quantification Integration

**Current Problem:**
- Quantifications stored separately: `Quantifications/{scan}/{user}/{quantId}.bin`
- Requires separate file load
- No spatial indexing

**Proposed Solution:**

**Option A: Embed in Point Tiles (RECOMMENDED)**
```go
// When quantification completes:
func UpdatePointTilesWithQuant(
    scanId string,
    quantId string,
    quantResults *protos.Quantification,
    s3 FileAccess,
) error {
    // 1. Load existing point tiles
    metadata := loadPointTileMetadata(s3, scanId)

    // 2. For each tile
    for zoom := 0; zoom < metadata.ZoomLevels; zoom++ {
        for _, tileKey := range getAllTileKeys(zoom) {
            // Load location tile
            tile := loadLocationTile(s3, scanId, tileKey)

            // Update quantification data for points in this tile
            for _, loc := range tile.Locations {
                if quantData := findQuantForPMC(quantResults, loc.Pmc); quantData != nil {
                    loc.QuantValues = convertToMap(quantData, quantResults.Labels)
                }
            }

            // Save updated tile
            saveLocationTile(s3, scanId, tileKey, tile)
        }
    }

    return nil
}
```

**Pros:**
- ✅ Single request for locations + quant data
- ✅ No separate quantification file load
- ✅ Spatial indexing applies to quant data

**Cons:**
- ❌ Tiles must update when quantification runs
- ❌ Multiple quantifications require versioning strategy

**Option B: Separate Quantification Tiles**
- Store quantification data in parallel tile structure
- Query: request location tile + quantification tile
- Pro: Tiles immutable, Con: Double requests

**Recommendation:** Start with Option A (embedded), optimize later if needed.

---

### Spatial Queries

**Backend MongoDB Index (for non-tile queries):**
```javascript
db.beamLocations.createIndex({
    scanId: 1,
    "location.2dsphere": "2dsphere"  // image_i, image_j as GeoJSON point
})
```

**Usage:**
```go
// Find all PMCs in image rectangle
func QueryPointsInViewport(
    scanId string,
    minI, maxI, minJ, maxJ float64,
    db *mongo.Database,
) ([]int32, error) {
    filter := bson.M{
        "scanId": scanId,
        "location": bson.M{
            "$geoWithin": bson.M{
                "$box": [][]float64{
                    {minI, minJ},  // Bottom-left
                    {maxI, maxJ},  // Top-right
                },
            },
        },
    }

    cursor := db.Collection("beamLocations").Find(context.TODO(), filter)
    // ... extract PMCs
}
```

**Use Cases:**
- ROI creation by drawing on image
- Quick "how many points in region?" queries
- Fallback for non-tiled legacy datasets

---

### ROI Storage Update

**Current:** ROI stores compressed location indices
**Proposed:** ROI stores tile coordinates + PMC list

```protobuf
message ROIItem {
    string id;
    string scanId;
    string name;

    // NEW: Tile-based storage
    repeated TileCoordinate tiles;      // Which tiles contain ROI points
    repeated int32 pmcs;                // Full list of PMCs

    // DEPRECATED (keep for backward compat)
    repeated int32 scanEntryIndexesEncoded;
}
```

**Benefits:**
- Faster loading: Request specific tiles only
- Spatial hint: Know which tiles to prefetch
- Backward compatible with existing ROIs

---

## Part 3: Unified Tiled Architecture

### How Images and Points Work Together

**Client Viewport Query:**
```
User views image region: x=[1000:2000], y=[500:1500] at zoom level 2
  ↓
Calculate required tiles:
  Image tiles: z2/x3_y1, z2/x3_y2, z2/x4_y1, z2/x4_y2
  Point tiles: z2/x3_y1, z2/x3_y2, z2/x4_y1, z2/x4_y2 (same!)
  ↓
Request both in parallel:
  GET /image-tiles/{scan}/{image}/2/3_1
  GET /image-tiles/{scan}/{image}/2/3_2
  GET /point-tiles/{scan}/2/3_1/locations
  GET /point-tiles/{scan}/2/3_2/locations
  ↓
Render image + overlay points in viewport
```

**Progressive Loading:**
1. User opens scan → Load zoom 4 (overview)
   - Image: 8×5 = 40 tiles @ 50 KB = **2 MB**
   - Points: 40 tiles @ 10 KB = **400 KB**
   - Total: **2.4 MB** (vs 2 GB + 10 GB before!)

2. User zooms to region → Load zoom 2
   - Visible: 10×10 tiles = 100 tiles
   - Image: 100 × 50 KB = **5 MB**
   - Points: 100 × 50 KB = **5 MB**
   - Total: **10 MB** for 25% of full detail

3. User zooms to full detail → Load zoom 0
   - Visible: 4×4 tiles = 16 tiles (zoomed to 1/16 of image)
   - Image: 16 × 50 KB = **800 KB**
   - Points: 16 × 200 KB = **3.2 MB**
   - Spectra: 16 × 2 MB = **32 MB**
   - Total: **36 MB** for full detail in viewport

**Key Insight:** Always load proportional to what's visible, not total dataset size.

---

### Frontend Integration

**Tile-Based Map Library:**
- Use **Leaflet.js** or **OpenLayers** (proven, mature)
- Custom tile layer for images
- Custom overlay layer for points

**Pseudo-Code:**
```javascript
// Leaflet.js integration

const map = L.map('map-container', {
  crs: L.CRS.Simple,  // Image coordinates, not lat/lon
  minZoom: 0,
  maxZoom: 4
});

// Image tile layer
const imageTileLayer = L.tileLayer(
  '/image-tiles/{scanId}/{imageId}/{z}/{x}_{y}?with-locations=false',
  {
    tileSize: 256,
    bounds: [[0, 0], [20000, 30000]]
  }
);
map.addLayer(imageTileLayer);

// Point overlay layer (custom)
const pointLayer = new L.GridLayer({
  tileSize: 256,
  createTile: function(coords, done) {
    const canvas = document.createElement('canvas');
    canvas.width = 256;
    canvas.height = 256;

    // Fetch point data for this tile
    fetch(`/point-tiles/${scanId}/${coords.z}/${coords.x}_${coords.y}/locations`)
      .then(resp => resp.arrayBuffer())
      .then(buffer => {
        const tile = PointDataTile.decode(buffer);

        // Render points on canvas
        const ctx = canvas.getContext('2d');
        tile.locations.forEach(loc => {
          const tileX = (loc.image_i % 256);
          const tileY = (loc.image_j % 256);

          ctx.fillStyle = getQuantColor(loc.quantValues);
          ctx.fillRect(tileX-2, tileY-2, 4, 4);  // 4px dot
        });

        done(null, canvas);
      });

    return canvas;
  }
});
map.addLayer(pointLayer);

// Interaction: click point to load spectrum
map.on('click', async (e) => {
  const coords = getTileCoords(e.latlng, map.getZoom());

  // Load spectra tile
  const spectraResp = await fetch(
    `/point-tiles/${scanId}/${coords.z}/${coords.x}_${coords.y}/spectra`
  );
  const spectraTile = SpectraTile.decode(await spectraResp.arrayBuffer());

  // Find closest point to click
  const closestPoint = findClosestPoint(e.latlng, pointLayer);
  const spectrum = spectraTile.spectra.find(s => s.pmc === closestPoint.pmc);

  // Display spectrum chart
  showSpectrumChart(spectrum);
});
```

**Libraries:**
- **Leaflet.js** (recommended): Lightweight, simple API
- **OpenLayers**: More features, heavier
- **MapLibre GL**: GPU-accelerated, great for large point clouds

---

### CDN and Caching Architecture

```
Client Browser
  ↓
CloudFront CDN (edge cache)
  ↓
S3 Bucket (origin)
  ↓
API Server (tile generation fallback)
```

**CloudFront Configuration:**
```yaml
# CloudFormation template
ImageTilesCacheBehavior:
  PathPattern: /image-tiles/*
  TargetOriginId: S3-DatasetsBucket
  CachePolicyId: CachingOptimized  # Managed policy
  AllowedMethods: [GET, HEAD, OPTIONS]
  Compress: true
  ViewerProtocolPolicy: redirect-to-https

PointTilesCacheBehavior:
  PathPattern: /point-tiles/*
  TargetOriginId: S3-DatasetsBucket
  CachePolicyId: CachingOptimized
  AllowedMethods: [GET, HEAD, OPTIONS]
  Compress: true
  ViewerProtocolPolicy: redirect-to-https
```

**Cache Headers:**
```go
// In tile endpoint handler
w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
w.Header().Set("ETag", calculateTileETag(z, x, y))
```

**Why This Works:**
- Tiles are immutable (content-addressed by z/x/y)
- 1-year cache TTL (essentially permanent)
- Edge caching reduces latency globally
- No cache invalidation needed (new imports = new tile coordinates)

---

## Implementation Plan

### Phase 1: Image Tiling Foundation (2-3 weeks)

**Goal:** Prove tile-based image serving works

**Tasks:**
1. **Create tile generation library**
   - `api/imagetiler/` package
   - Streaming TIFF reader (avoid loading full image)
   - Pyramid generator (zoom levels 0-4)
   - PNG encoder per tile
   - Estimated: 3-4 days

2. **Implement tile storage**
   - S3 path structure: `Images/{scan}/{image}/tiles/z{z}/x{x}_y{y}.png`
   - Metadata JSON writer
   - Estimated: 1 day

3. **Create tile REST endpoints**
   - `GET /image-tiles/{scan}/{image}/metadata`
   - `GET /image-tiles/{scan}/{image}/{z}/{x}/{y}`
   - ETag/caching support
   - Location: `api/endpoints/ImageTile.go`
   - Estimated: 2 days

4. **Update import pipeline**
   - Detect large images (> 4096×4096)
   - Call tile generator during import
   - Fallback to legacy for small images
   - Location: `api/dataimport/internal/output/imageTiler.go`
   - Estimated: 2 days

5. **Testing**
   - Unit tests for tile generator
   - Integration test with sample 10k×10k TIFF
   - Verify tile alignment and zoom levels
   - Estimated: 2 days

6. **Frontend spike**
   - Proof-of-concept Leaflet.js integration
   - Load tiles from new endpoint
   - Basic zoom/pan interaction
   - Estimated: 3 days (frontend team)

**Deliverable:** Large images served as tiles, viewable in browser

---

### Phase 2: Point Data Tiling (3-4 weeks)

**Goal:** Tile point data matching image tiles

**Tasks:**
1. **Design protobuf schema**
   - `PointDataTile` message
   - `LocationSummary` message
   - `SpectraTile` message (separate storage)
   - Location: `data-formats/api-messages/point-tiles.proto`
   - Estimated: 1 day

2. **Create point tiling library**
   - `api/pointdata/tiler.go`
   - Spatial indexing (map points to tiles)
   - Downsampling strategy per zoom level
   - Location + spectra file writers
   - Estimated: 4-5 days

3. **Implement tile REST endpoints**
   - `GET /point-tiles/{scan}/metadata`
   - `GET /point-tiles/{scan}/{z}/{x}/{y}/locations`
   - `GET /point-tiles/{scan}/{z}/{x}/{y}/spectra`
   - Location: `api/endpoints/PointTile.go`
   - Estimated: 2 days

4. **WebSocket batch query**
   - `PointTileBatchReq` / `PointTileBatchResp` messages
   - Handler to fetch multiple tiles efficiently
   - Location: `api/ws/handlers/pointTile.go`
   - Estimated: 2 days

5. **Update import pipeline**
   - Generate point tiles after experiment file creation
   - Parallel generation with image tiles
   - Estimated: 3 days

6. **MongoDB spatial index**
   - Create `beamLocations` collection
   - Insert point coordinates with 2dsphere index
   - Query helpers for viewport-based selection
   - Location: `api/dbCollections/collections.go`
   - Estimated: 2 days

7. **Testing**
   - Unit tests for spatial indexing
   - Integration test: Load 100k point dataset
   - Verify downsampling and tile assignment
   - Estimated: 3 days

8. **Frontend integration**
   - Point overlay layer in map viewer
   - Load points for visible tiles
   - Click interaction to load spectra
   - Estimated: 5 days (frontend team)

**Deliverable:** Point data loaded on-demand, correlated with image tiles

---

### Phase 3: Quantification Integration (2 weeks)

**Goal:** Embed quantification results in point tiles

**Tasks:**
1. **Extend protobuf schema**
   - Add `quantValues` map to `LocationSummary`
   - Support multiple quantification versions
   - Estimated: 1 day

2. **Tile update logic**
   - Function to update point tiles with quant results
   - Load → modify → save workflow for each tile
   - Location: `api/quantification/updateTiles.go`
   - Estimated: 3 days

3. **Update quantification pipeline**
   - After PIQUANT job completes, update point tiles
   - Notify clients via WebSocket (tiles updated)
   - Estimated: 2 days

4. **Versioning strategy**
   - Store multiple quantifications: `quant_{quantId}` field
   - Client selects active quantification
   - Estimated: 2 days

5. **Frontend integration**
   - Color-code points by quantification values
   - Display quant data on point hover
   - Estimated: 3 days (frontend team)

**Deliverable:** Quantification results visible on map without separate file load

---

### Phase 4: Beam Location Overlays (1-2 weeks)

**Goal:** Pre-generate tiles with beam location markers

**Tasks:**
1. **Overlay tile generator**
   - Function to load base tile + draw beam markers
   - Save to `tiles-with-locations/` directory
   - Location: `api/imagetiler/overlayGenerator.go`
   - Estimated: 3 days

2. **Update import pipeline**
   - After point tiles generated, create overlay tiles
   - Parallel processing for speed
   - Estimated: 2 days

3. **Update tile endpoint**
   - `?with-locations=true` serves from overlay directory
   - Estimated: 1 day

4. **Testing**
   - Verify marker positioning accuracy
   - Check alignment across zoom levels
   - Estimated: 2 days

**Deliverable:** Beam locations visible on image tiles without runtime overlay

---

### Phase 5: CDN and Optimization (1 week)

**Goal:** Deploy CDN for global low-latency access

**Tasks:**
1. **CloudFront distribution setup**
   - Create distribution pointing to S3
   - Configure cache behaviors for tile paths
   - Enable compression
   - Location: Infrastructure / CloudFormation
   - Estimated: 1 day

2. **Update API endpoints**
   - Return CDN URLs instead of direct S3
   - Serve tiles through CloudFront
   - Estimated: 1 day

3. **Cache header optimization**
   - Set long TTL for tiles (1 year)
   - Add `immutable` cache directive
   - Estimated: 1 day

4. **Performance testing**
   - Load testing with large datasets
   - Measure tile load times
   - Verify CDN hit rates
   - Estimated: 2 days

**Deliverable:** Production-ready tiled architecture with global CDN

---

### Phase 6: Migration and Cleanup (2 weeks)

**Goal:** Migrate existing datasets, remove legacy code

**Tasks:**
1. **Migration script**
   - CLI tool to retile existing datasets
   - Batch process all scans in S3
   - Location: `cmd/retile-datasets/main.go`
   - Estimated: 3 days

2. **Run migration**
   - Process production datasets
   - Validate tile integrity
   - Estimated: 3 days (with monitoring)

3. **Update frontend**
   - Remove legacy single-image code paths
   - Switch all views to tile-based
   - Estimated: 3 days (frontend team)

4. **Deprecate legacy endpoints**
   - Add deprecation warnings to `/images/{scan}/{filename}`
   - Set sunset date
   - Estimated: 1 day

5. **Documentation**
   - Update CLAUDE.md with tile architecture
   - Add tile endpoint documentation
   - Estimated: 2 days

**Deliverable:** All datasets using tile-based architecture

---

### Phase 7: Advanced Features (Future)

**Nice-to-Have Enhancements:**

1. **WebGL Point Rendering** (1 week)
   - Use MapLibre GL for GPU-accelerated point rendering
   - Handle 1M+ points smoothly
   - Location: Frontend

2. **Vector Tiles for Points** (1 week)
   - Use MVT (Mapbox Vector Tiles) format instead of protobuf
   - Better compression and standard tooling
   - Location: `api/pointdata/vectorTiles.go`

3. **Tile Pre-Warming** (3 days)
   - Background job to pre-generate all tiles on import
   - Upload to S3 before dataset marked ready
   - Location: `api/job/tileWarmer.go`

4. **Smart Caching** (1 week)
   - Track tile access patterns
   - Pre-fetch adjacent tiles on zoom
   - Client-side IndexedDB cache
   - Location: Frontend

5. **Multi-Channel TIFF Support** (1 week)
   - Tile RGBU images preserving all channels
   - Client-side channel mixing
   - Location: `api/imagetiler/multiChannel.go`

---

## Migration Strategy

### Backward Compatibility Approach

**Dual-Mode Operation:**
- Legacy datasets: Use existing `/images/` endpoint
- New datasets: Generate tiles on import
- Frontend: Detect tile availability, use appropriate loader

**Detection Logic:**
```go
// api/endpoints/Image.go

func GetImage(params) (*s3.GetObjectOutput, ...) {
    // Check if tiled version exists
    metadataPath := fmt.Sprintf("Images/%s/%s/metadata.json", scanId, imageId)

    if params.Svcs.FS.ObjectExists(metadataPath, params.Svcs.Config.DatasetsBucket) {
        // Tiled version available → redirect or serve preview
        return handleTiledImage(params)
    }

    // Fall back to legacy single-file serving
    return handleLegacyImage(params)
}
```

---

### Migration Timeline

**Months 1-2:** Implement Phase 1-2 (image + point tiling)
**Month 3:** Implement Phase 3-4 (quant integration + overlays)
**Month 4:** Deploy Phase 5 (CDN), begin migration (Phase 6)
**Month 5:** Complete migration, cleanup legacy code
**Month 6+:** Advanced features (Phase 7)

---

### Rollback Plan

**If Issues Arise:**
1. Frontend toggle: `ENABLE_TILED_VIEWER=false`
2. API continues serving legacy endpoints
3. New tile generation disabled in import pipeline
4. Existing tiles remain in S3 (no data loss)

**Risk Mitigation:**
- Deploy to staging environment first
- Gradual rollout: 10% → 50% → 100% of users
- Monitor error rates and load times
- A/B test performance metrics

---

## Performance Projections

### Current vs. Tiled Architecture

**Scenario: 30k × 20k image, 1M points**

| Metric | Current | Tiled | Improvement |
|--------|---------|-------|-------------|
| **Initial Load (Overview)** |
| Image data | 2 GB | 2 MB | **1000x faster** |
| Point data | 10 GB | 400 KB | **25,000x faster** |
| Total | 12 GB | 2.4 MB | **5000x faster** |
| **Zoomed to 25% of Image** |
| Image data | 2 GB | 5 MB | **400x faster** |
| Point data | 10 GB | 5 MB | **2000x faster** |
| Total | 12 GB | 10 MB | **1200x faster** |
| **Full Detail in Viewport (1/16 of image)** |
| Image data | 2 GB | 800 KB | **2500x faster** |
| Point data | 10 GB | 35 MB | **285x faster** |
| Total | 12 GB | 36 MB | **333x faster** |

---

### Storage Impact

**Example: 30k × 20k image with 1M points**

| Component | Size | Notes |
|-----------|------|-------|
| **Images** |
| Original TIFF | 2 GB | Kept as source |
| Base tiles (z0-z4) | 465 MB | All zoom levels |
| Overlay tiles (with locations) | 465 MB | Pre-rendered |
| Total images | 2.93 GB | **1.5x original** |
| **Point Data** |
| Original dataset.bin | 10 GB | Legacy format |
| Location tiles (z0-z4) | 150 MB | Coordinates only |
| Spectra tiles (z0-z4) | 8 GB | Full spectra |
| Total points | 8.15 GB | **0.8x original** |
| **Grand Total** | 11.1 GB vs 12 GB | **~Same storage, 1000x faster** |

**Key Insight:** Storage cost is roughly the same, but access patterns are vastly improved.

---

### Network Transfer Savings

**User Session Example:**
1. Open scan → Load overview
   - Old: 12 GB
   - New: 2.4 MB
   - **Savings: 99.98%**

2. Zoom to region (10 interactions)
   - Old: 120 GB (reloads full data)
   - New: 100 MB (incremental tiles)
   - **Savings: 99.92%**

3. View full detail in small region (5 interactions)
   - Old: 60 GB
   - New: 180 MB
   - **Savings: 99.70%**

**Total Session:**
- Old: **192 GB transferred**
- New: **282 MB transferred**
- **Savings: 99.85%**

---

### Scalability Limits

**With Tiled Architecture, we can handle:**
- Images up to **100k × 100k pixels** (10 gigapixels)
- Point datasets with **10M+ points**
- Hundreds of concurrent users
- Global access with <100ms tile load times (via CDN)

**Bottlenecks Eliminated:**
- ✅ No more full-file downloads
- ✅ No more memory pressure on API servers
- ✅ No more client-side rendering limits
- ✅ No more spatial query slowdowns

---

## Technical Decisions and Reasoning

### Why Tile Size = 256×256?

**Reasoning:**
- **Industry standard:** Used by Google Maps, OpenStreetMap, etc.
- **Cache-friendly:** ~50 KB PNG fits easily in browser/CDN cache
- **Network optimal:** Small enough for fast HTTP requests, large enough to avoid overhead
- **GPU-friendly:** Power-of-2 dimensions work well with WebGL textures

**Alternatives Considered:**
- 512×512: Larger files, fewer requests, but slower initial loads
- 128×128: More requests, more overhead, no real benefit

---

### Why Pre-Generate Overlays vs. Dynamic?

**Decision:** Pre-generate

**Reasoning:**
- **Latency:** Pre-rendering eliminates 100ms+ per tile request
- **CPU:** API servers don't waste cycles drawing markers
- **CDN:** Immutable tiles cache perfectly
- **Storage:** Cost is negligible (~465 MB for 2 GB image)

**Trade-off:** Must regenerate if beam locations change (rare after import)

---

### Why Separate Spectra Tiles?

**Decision:** Store locations and spectra separately

**Reasoning:**
- **Size:** Spectra are 99% of point data volume (16 KB vs 40 bytes)
- **Use case:** Users often browse locations without viewing spectra
- **Bandwidth:** Don't transfer spectra unless clicked
- **Cache:** Location tiles cache hot, spectra cache cold

**Alternative Considered:** Inline spectra → 400x larger tiles, impractical

---

### Why Embed Quantification in Location Tiles?

**Decision:** Store quant values in `LocationSummary`

**Reasoning:**
- **Simplicity:** Single request for all point data
- **Use case:** Quantification is primary visualization mode
- **Size:** Quant data is small (~20 floats per point = 80 bytes)
- **Performance:** Eliminates separate quantification file load

**Trade-off:** Tiles must update when quantification runs (acceptable for improved UX)

---

### Why Downsample Points at Lower Zoom Levels?

**Decision:** Reduce point density at overview zoom levels

**Reasoning:**
- **Visual:** 1M points on screen → 1 pixel per point anyway
- **Bandwidth:** Don't transfer invisible points
- **Performance:** Client rendering faster with fewer points
- **Progressive:** Matches progressive image loading pattern

**Example:** Zoom 4 (overview) shows 4k points instead of 1M, visually identical

---

### Why MongoDB Spatial Index + Tiles?

**Decision:** Use both MongoDB 2dsphere index AND tile storage

**Reasoning:**
- **Tiles:** Fast viewport queries for tiled data
- **MongoDB:** Flexible queries (ROI by property, spatial joins)
- **Redundancy:** MongoDB as source-of-truth, tiles as cache
- **Migration:** Legacy data still queryable during migration

**Not Exclusive:** Use tiles for viewer, MongoDB for analysis

---

## Open Questions / Future Discussions

1. **Multi-Quantification Strategy:**
   - Should tiles store all quantifications or only active one?
   - How to handle dozens of quant versions per dataset?
   - Potential: Separate quant tiles? Client-side merging?

2. **Vector Tiles for Points:**
   - Migrate to MVT (Mapbox Vector Tiles) standard?
   - Better compression, wider tool support
   - But: Requires different backend libraries

3. **WebAssembly for Tiling:**
   - Generate tiles client-side from raw data?
   - Reduces server load, but increases client requirements
   - Good for user uploads?

4. **Real-Time Data Streaming:**
   - For live instrument data during rover operations
   - Tiles generated incrementally as points arrive?
   - WebSocket streaming of partial tiles?

5. **3D Tile Support:**
   - Terrain models, 3D point clouds
   - Use 3D Tiles spec (Cesium)?
   - Out of scope for now, but future-proofing?

---

## Summary and Recommendations

### Core Insights

1. **Both images and point data need tiling** because they're correlated (points have image coordinates)
2. **Unified tile grid** enables perfect synchronization between layers
3. **Storage cost is negligible** (~1.5x) for massive performance gains (1000x+)
4. **Progressive loading** matches user workflow (overview → zoom → detail)
5. **CDN caching** makes global access fast and cheap

---

### Recommended Approach

**Phase 1-2 (Essential):**
- Implement image tiling + point tiling
- Prove scalability with large datasets
- Deploy to production for new imports

**Phase 3-4 (High Value):**
- Integrate quantification into tiles
- Pre-generate beam location overlays
- Eliminate separate file loads

**Phase 5 (Production Ready):**
- Deploy CDN for global performance
- Optimize cache headers

**Phase 6 (Cleanup):**
- Migrate existing datasets
- Deprecate legacy endpoints

---

### Key Metrics to Track

- **Initial load time:** < 3 seconds (vs. minutes currently)
- **Tile load time:** < 100ms per tile
- **CDN hit rate:** > 90%
- **Storage growth:** < 2x original dataset size
- **User-perceived performance:** "instant" zoom/pan

---

### Next Steps

1. **Review this document** with team (architecture, frontend, DevOps)
2. **Prioritize phases** based on business needs
3. **Allocate resources** (2 backend devs, 1 frontend dev, DevOps support)
4. **Create detailed tickets** for Phase 1 tasks
5. **Set up staging environment** with sample large dataset
6. **Begin Phase 1 implementation**

---

## Conclusion

The proposed tile-based architecture scales PIXLISE from rover-scale (thousands of points, megabyte images) to terrestrial-scale (millions of points, gigabyte images) with:
- **1000x improvement** in initial load times
- **Same storage cost** as current architecture
- **Progressive loading** matching user workflows
- **Global performance** via CDN caching
- **Future-proof** for even larger datasets

This is a proven pattern (Google Maps, OpenStreetMap, etc.) adapted to scientific data analysis. The implementation is substantial but achievable in 4-6 months with proper resourcing.

**Recommendation: Proceed with implementation, starting with Phase 1 (image tiling) to validate approach.**
