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
	"image/png"
	"os"
	"strings"
)

func ReadImageFile(path string) (image.Image, error) {
	// Load the full context image from test data
	imgbytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(imgbytes))
	if err != nil {
		return nil, err
	}
	return img, nil
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
