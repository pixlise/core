package imageedit

import (
	"image"
	"image/color"

	protos "github.com/pixlise/core/v3/generated-protos"
	"golang.org/x/image/draw"
)

func MarkLocations(img image.Image, locations []*protos.Coordinate2D, markColour color.Color, imgMatchTransform *protos.ImageMatchTransform) image.Image {
	// Copy the image data to output image (in greyscale)
	bounds := img.Bounds()

	outImage := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	/*if img.ColorModel() == color.GrayModel {
		outImage = image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	}*/

	draw.Draw(outImage, outImage.Bounds(), img, bounds.Min, draw.Src)

	// Run through all locations & set a white pixel where they are in the context image
	for _, loc := range locations {
		if loc != nil && loc.I > 0 && loc.J > 0 {
			i := loc.I
			j := loc.J

			if imgMatchTransform != nil {
				i *= imgMatchTransform.XScale
				i -= imgMatchTransform.XOffset

				j *= imgMatchTransform.YScale
				j -= imgMatchTransform.YOffset
			}

			outImage.Set(int(i+0.5), int(j+0.5), markColour)
		}
	}

	// We're done with this image
	return outImage
}
