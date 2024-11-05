// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package utils

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"os"
	"strings"

	protos "github.com/pixlise/core/v4/generated-protos"
)

// Returns width, height and error
func ReadImageDimensions(imageName string, imgBytes []byte) (uint32, uint32, error) {
	// Try to read the image
	img, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		// Failed, maybe it's a RGBU TIF image
		upperName := strings.ToUpper(imageName)
		if strings.Contains(err.Error(), "sample format") && (strings.Contains(upperName, "VIS_") || strings.Contains(upperName, "MSA_")) && strings.HasSuffix(upperName, ".TIF") {
			// We can't read these tif files, but it's an RGBU image, and they have a known resolution - the same as our MCC images
			return 752, 580, nil
		}

		return 0, 0, err
	}
	return uint32(img.Bounds().Dx()), uint32(img.Bounds().Dy()), nil
}

func ImagesEqual(aPath, bPath string) error {
	// Load the full context image from test data
	imgbytes, err := os.ReadFile(aPath)
	if err != nil {
		return err
	}

	a, _, err := image.Decode(bytes.NewReader(imgbytes))
	if err != nil {
		return err
	}

	imgbytes, err = os.ReadFile(bPath)
	if err != nil {
		return err
	}

	b, _, err := image.Decode(bytes.NewReader(imgbytes))
	if err != nil {
		return err
	}

	if a.Bounds().Dx() != b.Bounds().Dx() || a.Bounds().Dy() != b.Bounds().Dy() {
		return fmt.Errorf("image bounds not equal: %+v, %+v", a.Bounds(), b.Bounds())
	}

	errs := ""
	for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
		for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
			aPixel := a.At(x, y)
			bPixel := b.At(x, y)
			if aPixel != bPixel {
				errs += fmt.Sprintf("image pixels at %v,%v not equal: %+v, %+v\n", x, y, aPixel, bPixel)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf(errs)
	}

	return nil
}

func WritePNGImageFile(pathPrefix string, img image.Image) error {
	fileName := pathPrefix
	if !strings.HasSuffix(fileName, ".png") {
		fileName += ".png"
	}

	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	png.Encode(f, img)
	return nil
}

func MakeScanImage(
	imgPath string,
	fileSize uint32,
	source protos.ScanImageSource,
	purpose protos.ScanImagePurpose,
	associatedScanIds []string,
	originScanId string,
	originImageURL string,
	matchInfo *protos.ImageMatchTransform,
	width uint32,
	height uint32) *protos.ScanImage {
	result := &protos.ScanImage{
		ImagePath: imgPath,

		Source:   source,
		Width:    width,
		Height:   height,
		FileSize: fileSize,
		Purpose:  purpose,

		AssociatedScanIds: associatedScanIds,
		OriginScanId:      originScanId,
		OriginImageURL:    originImageURL,

		MatchInfo: matchInfo,
	}

	return result
}
