package imgFormat

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
)

type uncHeader struct {
	Word1 uint16
	Word2 uint16
}

type uncHexapodMetadata struct {
	Word1 uint16
	Word2 uint16
	Word3 uint16
	Word4 uint16
	Word5 uint16
	Word6 uint16
	Word7 uint16
}

type uncHeaderFields struct {
	DAC_OFFSET uint16
	DAC_gain   uint16
}

type uncHeaderMetadata struct {
	INTEGRATION_TIME      uint8
	COMPRESSION           uint8
	ROI                   uint8
	JPEG_QUALITY          uint8
	COMPRESSION_THRESHOLD uint8
	INFO                  uint8

	VALID  uint16
	STATUS uint16

	CODE_START uint32
	CODE_END   uint32

	SUB_TIMESTAMP uint16

	TIMESTAMP uint32

	H    uint16
	W    uint16
	IMOD uint16
}

func readUNCFileHeaders(data []byte) (uncHeaderFields, uncHeaderMetadata, *bytes.Reader, error) {
	buf := bytes.NewReader(data)
	var h uncHeader
	var hV uncHeaderFields
	var hM uncHeaderMetadata

	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return hV, hM, nil, err
	}

	// Check if we need to read the hexapod header
	if h.Word2 == 65535 {
		var hexapod uncHexapodMetadata

		err = binary.Read(buf, binary.LittleEndian, &hexapod)
		if err != nil {
			return hV, hM, nil, err
		}

		// We don't seem to need this? But read the next 2 bytes because they're a new header
		err = binary.Read(buf, binary.LittleEndian, &h)
		if err != nil {
			return hV, hM, nil, err
		}
	}

	// Set the first 2 values
	hV.DAC_OFFSET = h.Word1
	hV.DAC_gain = h.Word2

	// Read the header itself now
	err = binary.Read(buf, binary.LittleEndian, &hM)
	if err != nil {
		return hV, hM, nil, err
	}

	return hV, hM, buf, err
}

func getUNCEmbeddedFormat(compression uint8) string {
	imageFormat := ""
	if compression == 0 {
		imageFormat = "unc"
	} else if compression == 1 {
		imageFormat = "cen"
	} else if compression == 2 {
		imageFormat = "roi"
	} else if compression == 3 {
		imageFormat = "jp0"
	} else if compression == 4 {
		imageFormat = "nonStlOb"
	} else if compression == 32 {
		imageFormat = "CC"
	} else if compression == 33 {
		imageFormat = "TRN"
	} else if compression == 34 {
		imageFormat = "sli"
	}

	return imageFormat
}

func ReadUNCFile(data []byte) (int, int, []byte, error) {
	_, hM, buf, err := readUNCFileHeaders(data)

	if err != nil {
		return 0, 0, []byte{}, err
	}

	imageFormat := getUNCEmbeddedFormat(hM.COMPRESSION)

	if imageFormat == "jp0" {
		img, err := jpeg.Decode(buf)
		if err != nil {
			return 0, 0, []byte{}, fmt.Errorf("Failed to read jpeg: %v", err)
		}

		rect := img.Bounds()
		decodedImg := image.NewRGBA(rect)
		draw.Draw(decodedImg, rect, img, rect.Min, draw.Src)

		return rect.Dx(), rect.Dy(), decodedImg.Pix, nil
	}

	return 0, 0, []byte{}, fmt.Errorf("Unsupported image type: %v", imageFormat)
}

func ExtractJPGFromUNCFile(data []byte, outPath string) error {
	_, hM, buf, err := readUNCFileHeaders(data)
	if err != nil {
		return err
	}

	if hM.COMPRESSION != 3 {
		return fmt.Errorf("Unsupported image type: %v", getUNCEmbeddedFormat(hM.COMPRESSION))
	}

	file, err := os.Create(outPath)
	if err != nil {
		return err
	}

	_ /*written*/, err = buf.WriteTo(file)

	return err
}
