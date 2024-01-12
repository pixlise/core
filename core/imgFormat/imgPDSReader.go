package imgFormat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Reading IMG files from PDS
// We read a very strict subset of what's possible.
// References:
// https://planetarydata.jpl.nasa.gov/img/data/mars2020/mars2020_mastcamz_sci_calibrated/document/mastcamz_derived_product_sis_v1.3.pdf
// https://pds-geosciences.wustl.edu/m2020/urn-nasa-pds-mars2020_mission/document_camera/Mars2020_Camera_SIS.pdf
// https://www-mipl.jpl.nasa.gov/external/VICAR_file_fmt.pdf
// https://www-mipl.jpl.nasa.gov/vicar_os/v1.0/vicar-docs/VICAR_guide_1.0.pdf
//
// IMG2PNG was recommended on some forums but this page
// seems really blacklisted... http://bjj.mmedia.is/utils/img2png/

func findVICAR(bytes []byte) (int, error) {
	// Read the first line, should be our ODL_VERSION_ID
	line := ""
	fields := map[string]string{}
	lineNo := 0

	for _, b := range bytes {
		if b == '\n' || b == '\r' {
			if lineNo == 0 && !strings.HasPrefix(line, "ODL_VERSION_ID") {
				return 0, errors.New("Expected to start with ODL_VERSION_ID")
			}

			if strings.HasPrefix(line, "^IMAGE") {
				break
			}

			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				fields[strings.Trim(parts[0], " ")] = strings.Trim(parts[1], " ")
			}

			line = ""
			lineNo++
			continue
		}

		line = line + string(b)
	}

	// Check what we expect...
	if ver, ok := fields["ODL_VERSION_ID"]; !ok {
		return 0, errors.New("ODL_VERSION_ID not found")
	} else {
		if ver != "ODL3" {
			return 0, errors.New("Unexpected ODL_VERSION_ID: " + ver)
		}
	}

	if recType, ok := fields["RECORD_TYPE"]; !ok {
		return 0, errors.New("RECORD_TYPE not found")
	} else {
		if recType != "FIXED_LENGTH" {
			return 0, errors.New("Unexpected RECORD_TYPE: " + recType)
		}
	}

	recBytesStr, ok := fields["RECORD_BYTES"]
	if !ok {
		return 0, errors.New("RECORD_BYTES not found")
	}
	labelRecsStr, ok := fields["LABEL_RECORDS"]
	if !ok {
		return 0, errors.New("LABEL_RECORDS not found")
	}

	recBytes, err := strconv.Atoi(recBytesStr)
	if err != nil {
		return 0, fmt.Errorf("Invalid RECORD_BYTES: %v", err)
	}
	labelRecs, err := strconv.Atoi(labelRecsStr)
	if err != nil {
		return 0, fmt.Errorf("Invalid LABEL_RECORDS: %v", err)
	}

	return recBytes * labelRecs, nil
}

func readVICARLabel(bytes []byte, vicarPos int64) (string, error) {
	// Hop to the VICAR spec, and assume there's at least some bytes to start with
	if len(bytes) < int(vicarPos)+50 {
		return "", errors.New("VICAR header seems to be missing")
	}

	// Read the VICAR line, we're expecting the LBLSIZE first, then we'll know how big this is
	lblStartBytes := string(bytes[vicarPos : vicarPos+50])

	lblSizeName := "LBLSIZE="
	if !strings.HasPrefix(lblStartBytes, lblSizeName) {
		return "", fmt.Errorf("Expected VICAR header to start with LBLSIZE, got %v", lblStartBytes)
	}

	lblSizeStr := lblStartBytes[len(lblSizeName):]
	pos := strings.Index(lblSizeStr, " ")

	if pos == -1 {
		return "", errors.New("Failed to read VICAR LBLSIZE")
	}

	lblSizeStr = lblSizeStr[0:pos]

	lblSize, err := strconv.Atoi(lblSizeStr)
	if err != nil || lblSize <= 200 { // expect SOME data...
		return "", fmt.Errorf("Read invalid VICAR label size %v. Error: %v", lblSizeStr, err)
	}

	// Check that we haven't reached EOF
	if len(bytes) < int(vicarPos)+lblSize {
		return "", errors.New("File seems truncated, VICAR label not complete")
	}

	vicarLabel := bytes[vicarPos : vicarPos+int64(lblSize)]

	// Find the first 0
	result := string(vicarLabel)
	for c := 0; c < len(vicarLabel); c++ {
		if vicarLabel[c] == 0 {
			result = result[0:c]
			break
		}
	}

	return result, nil
}

// Returns lblSize, width, height, pixelBytes, err
func parseVICARLabel(vicarLabel string) (int, int, int, int, int, error) {
	// Expecting something like: LBLSIZE=19776           FORMAT='HALF'  TYPE='IMAGE'  BUFSIZ=3296  DIM=3  EOL=0  RECSIZE=3296  ORG='BSQ'  NL=1200  NS=1648  NB=3  N1=1648  N2=1200
	// We want to read NL (lines), NS (samples) and FORMAT to work out the image dimensions
	vicarItems := strings.Split(vicarLabel, " ")

	lblSize := 0
	width := 0
	height := 0
	pixelBytes := 0
	channels := 0
	var err error

	for _, item := range vicarItems {
		err = nil
		if strings.HasPrefix(item, "LBLSIZE=") {
			lblSize, err = strconv.Atoi(item[8:])
		}
		if strings.HasPrefix(item, "NL=") {
			height, err = strconv.Atoi(item[3:])
		}
		if strings.HasPrefix(item, "NS=") {
			width, err = strconv.Atoi(item[3:])
		}
		if strings.HasPrefix(item, "NB=") {
			channels, err = strconv.Atoi(item[3:])
		}
		if strings.HasPrefix(item, "EOL=") {
			if item[4:] != "0" {
				return 0, 0, 0, 0, 0, errors.New("Expected EOL=0")
			}
		}
		if strings.HasPrefix(item, "NBB=") {
			if item[4:] != "0" {
				return 0, 0, 0, 0, 0, errors.New("Expected NBB=0")
			}
		}
		if strings.HasPrefix(item, "NLB=") {
			if item[4:] != "0" {
				return 0, 0, 0, 0, 0, errors.New("Expected NLB=0")
			}
		}
		if strings.HasPrefix(item, "BITS=") {
			bits := item[5:]
			if bits != "12" {
				return 0, 0, 0, 0, 0, fmt.Errorf("Unexpected BITS %v", bits)
			}
		}
		if strings.HasPrefix(item, "ORG=") {
			org := item[4:]
			if org != "'BSQ'" {
				return 0, 0, 0, 0, 0, fmt.Errorf("Unexpected ORG %v", org)
			}
		}
		if strings.HasPrefix(item, "DIM=") {
			if item[4:] != "3" {
				return 0, 0, 0, 0, 0, errors.New("Expected DIM=3")
			}
		}
		if strings.HasPrefix(item, "TYPE=") {
			txt := item[5:]
			if txt != "'IMAGE'" {
				return 0, 0, 0, 0, 0, fmt.Errorf("Unexpected TYPE %v", txt)
			}
		}
		if strings.HasPrefix(item, "FORMAT=") {
			format := item[7:]

			if format != "'HALF'" && format != "'WORD'" {
				return 0, 0, 0, 0, 0, fmt.Errorf("Unsupported VICAR pixel format: %v", format)
			}
			pixelBytes = 2
		}

		if err != nil {
			return 0, 0, 0, 0, 0, fmt.Errorf("Invalid VICAR label item: %v. Error: %v", item, err)
		}
	}

	if width == 0 || height == 0 || pixelBytes == 0 {
		return 0, 0, 0, 0, 0, errors.New("Invalid VICAR label, missing expected fields")
	}

	return lblSize, width, height, pixelBytes, channels, nil
}

func readVICARImage(bytes []byte, imgOffset int64, imgSize int64) ([]byte, error) {
	buff := make([]byte, imgSize)
	readBytes := copy(buff, bytes[imgOffset:])

	if readBytes != int(imgSize) {
		return []byte{}, fmt.Errorf("Failed to read VICAR image of %v bytes, got %v bytes", imgSize, readBytes)
	}

	return buff, nil
}

func ReadIMGFile(bytes []byte) (int, int, []byte, error) {
	vicarPos, err := findVICAR(bytes)
	if err != nil {
		return 0, 0, nil, err
	}

	vicarLabel, err := readVICARLabel(bytes, int64(vicarPos))
	if err != nil {
		return 0, 0, nil, err
	}

	// Now figure out the image format
	lblSize, width, height, bytesPP, channels, err := parseVICARLabel(vicarLabel)
	if err != nil {
		return 0, 0, nil, err
	}

	if channels != 1 && channels != 3 {
		return 0, 0, nil, fmt.Errorf("Expected 1 or 3 channels in image")
	}

	// Hop to the image bytes and read them
	imgPos := vicarPos + lblSize
	imgSize := width * height * channels * bytesPP
	imageBytes, err := readVICARImage(bytes, int64(imgPos), int64(imgSize))
	if err != nil {
		return 0, 0, nil, err
	}

	// We're expecting 2 bytes per pixel, turn this into RGBA 1 byte
	result := make([]byte, width*height*4)

	// Expecting BSQ, so R, G and B separately stored each as 2 byte values
	bandSize := width * height * bytesPP
	maxPixel := float64(1 << 12)

	// Should be BSQ, lines (of RECSIZE) containing samples
	// So first coordinate accessed should be at 0, then 2, then 4 etc to RECSIZE

	for ch := 0; ch < channels; ch++ {
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				offset := ch*bandSize + (y*width+x)*bytesPP
				// because we're reading 2 byte pixel values:
				//offset *= 2

				readVal := (uint16(imageBytes[offset]) << 8) | uint16(imageBytes[offset+1])

				// Assumption: 12 bits per pixel read (see BITS in VICAR parser)
				saveValue := byte(255 * float64(readVal) / maxPixel)

				writeOffset := (y*width + x) * 4
				result[writeOffset+ch] = byte(saveValue)

				// if we're reading 1 channel, write to G and B
				if channels == 1 {
					result[writeOffset+1] = saveValue
					result[writeOffset+2] = saveValue
					result[writeOffset+3] = 255
				} else if ch == 2 {
					// Write out the alpha value
					result[writeOffset+3] = 255
				}
			}
		}
	}

	return width, height, result, nil
}
