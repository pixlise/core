package imageedit

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
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
