package imageedit

import (
	"image"

	"golang.org/x/image/draw"
)

func ScaleImage(img image.Image, newWidth int) image.Image {
	bounds := img.Bounds()

	// We want it to be a max of newWidth across, preserving the aspect ratio
	// we calculate the height here
	w := newWidth
	h := int(float32(bounds.Max.Y) / float32(bounds.Max.X) * float32(w))

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.ApproxBiLinear.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)

	return dst
}
