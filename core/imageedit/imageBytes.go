package imageedit

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

func GetImageBytes(img image.Image, imgFormat string) ([]byte, error) {
	var err error
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)

	if imgFormat == "png" {
		err = png.Encode(writer, img)
	} else if imgFormat == "jpeg" {
		err = jpeg.Encode(writer, img, &jpeg.Options{Quality: 90})
	} else {
		err = fmt.Errorf("unexpected image format: %v when applying image modifications", imgFormat)
	}

	if err != nil {
		return nil, err
	}

	err = writer.Flush()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func MakeImageFromRGBA(width int, height int, data []byte) image.Image {
	i := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			idx := (y*width + x) * 4
			r := data[idx]
			g := data[idx+1]
			b := data[idx+2]
			a := data[idx+3]
			i.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}

	return i
}
