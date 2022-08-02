// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"strings"
)

func ReadImageFile(path string) (image.Image, error) {
	// Load the full context image from test data
	imgbytes, err := ioutil.ReadFile(path)
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
	imgbytes, err := ioutil.ReadFile(aPath)
	if err != nil {
		return err
	}

	a, _, err := image.Decode(bytes.NewReader(imgbytes))
	if err != nil {
		return err
	}

	imgbytes, err = ioutil.ReadFile(bPath)
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
