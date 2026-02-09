package wsHelpers

import (
	"math"

	protos "github.com/pixlise/core/v4/generated-protos"
)

func BuildImagePyramidLevels(imageWidth uint32, imageHeight uint32, tileSize uint32) *protos.ImagePyramid {
	// Calculate number of zoom levels
	maxDim := float64(imageWidth)
	if imageHeight > imageWidth {
		maxDim = float64(imageHeight)
	}
	numLevels := int(math.Ceil(math.Log2(maxDim/float64(tileSize)))) + 1

	// Overall bounds (using page 0 dimensions)
	bounds := &protos.AABB{
		Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
		Max: &protos.Coordinate3D{X: float32(imageWidth), Y: float32(imageHeight), Z: 0},
	}

	// Build layers (one per zoom level)
	layers := make([]*protos.ImagePyramidLayer, numLevels)
	for level := 0; level < numLevels; level++ {
		// Calculate dimensions at this level
		scale := math.Pow(2, float64(numLevels-level-1))
		levelWidth := int(math.Ceil(float64(imageWidth) / scale))
		levelHeight := int(math.Ceil(float64(imageHeight) / scale))

		// Calculate number of tiles at this level
		tilesX := int(math.Ceil(float64(levelWidth) / float64(tileSize)))
		tilesY := int(math.Ceil(float64(levelHeight) / float64(tileSize)))

		// Create tile summaries (with points=0, polygons=0 for now)
		tiles := make([]*protos.ImageTileSummary, 0, tilesX*tilesY)
		for y := 0; y < tilesY; y++ {
			for x := 0; x < tilesX; x++ {
				// Calculate tile bounds
				tileX := float32(x * int(tileSize))
				tileY := float32(y * int(tileSize))
				tileW := float32(tileSize)
				tileH := float32(tileSize)

				// Clamp to level dimensions
				if tileX+tileW > float32(levelWidth) {
					tileW = float32(levelWidth) - tileX
				}
				if tileY+tileH > float32(levelHeight) {
					tileH = float32(levelHeight) - tileY
				}

				tiles = append(tiles, &protos.ImageTileSummary{
					Bounds: &protos.AABB{
						Min: &protos.Coordinate3D{X: tileX, Y: tileY, Z: 0},
						Max: &protos.Coordinate3D{X: tileX + tileW, Y: tileY + tileH, Z: 0},
					},
					Points:   0, // TODO: Will be populated when overlay data added
					Polygons: 0, // TODO: Will be populated when overlay data added
				})
			}
		}

		layers[level] = &protos.ImagePyramidLayer{
			Bounds: &protos.AABB{
				Min: &protos.Coordinate3D{X: 0, Y: 0, Z: 0},
				Max: &protos.Coordinate3D{X: float32(levelWidth), Y: float32(levelHeight), Z: 0},
			},
			Tiles:     tiles,
			TilesWide: uint32(tilesX),
			TilesHigh: uint32(tilesY),
		}
	}

	// Image prefix (base path for tile files)
	// Points to the image directory containing all pages

	return &protos.ImagePyramid{
		Bounds:  bounds,
		Pyramid: layers,
		//ImagePrefixes: []string{imagePrefix},
		TileSize: uint32(tileSize),
	}
}
