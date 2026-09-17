package bruker

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"

	brukerreader "github.com/pixlise/core/v4/api/dataimport/internal/converters/bruker/bruker-reader"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/core/imageedit"
)

func (b Bruker) readImage(suffix string, importPath string, item *brukerreader.HspyItem) (string, int, int, int, error) {
	axes, err := readAxes(item.Axes)
	if err != nil {
		return "", 0, 0, 0, err
	}

	// For now we only read single channel images in
	channelCount := int64(0)
	if f, err := getMetaField(item.OriginalMetadata, "DSP Configuration", "ChannelCount"); err == nil {
		channelCount = f.(int64)
	} else {
		return "", 0, 0, 0, err
	}

	if channelCount != 1 {
		return "", 0, 0, 0, fmt.Errorf("Expected 1 channel, got %v", channelCount)
	}

	sampleName := ""
	if f, err := getMetaField(item.Metadata, "Sample", "name"); err == nil {
		sampleName = f.(string)
	} else {
		return "", 0, 0, 0, err
	}

	// Try some data types
	var img image.Image

	bytesPerChannel := 1
	if uint16data, ok := item.Data.([]uint16); ok {
		img = imageedit.MakeMonochromeImage(axes.width, axes.height, uint16data)
		bytesPerChannel = 2
	} else if uint8data, ok := item.Data.([]uint8); ok {
		img = imageedit.MakeMonochromeImage(axes.width, axes.height, uint8data)
	} else {
		return "", 0, 0, 0, fmt.Errorf("Unexpected image data type: %v", reflect.TypeOf(item.Data))
	}

	pngPath := filepath.Join(importPath, fmt.Sprintf("%v_%v.png", sampleName, suffix))
	if err := b.writeImage(pngPath, img); err != nil {
		return "", 0, 0, 0, err
	}

	return pngPath, bytesPerChannel, axes.width, axes.height, nil
}

func (b Bruker) readAlignedImages(
	scanId string,
	images []*brukerreader.HspyItem,
	importPath string,
	spectrumWidth int, spectrumHeight int,
	maxPMC int32) ([]dataConvertModels.MatchedAlignedImageMeta, error) {
	// Here we keep track of widths, heights and data format. If the last 3 match as we go along we
	// generate an RGB image out of them too
	lastWidths := []int{}
	lastHeights := []int{}
	lastBytesPerChannels := []int{}
	lastSignals := []string{}

	alignedImages := []dataConvertModels.MatchedAlignedImageMeta{}

	var width, height int

	nextPMC := maxPMC + 1

	for itemIdx, item := range images {
		signal, _, err := readBasicItemMeta(item)
		if err != nil {
			return alignedImages, err
		}

		if signal != "image" {
			return alignedImages, fmt.Errorf("Expected signal image, got %v", signal)
		}

		suffix := fmt.Sprintf("aligned-%v", itemIdx)

		// This is an "image" file, convert it to PNG so we can display/interpret it like other images
		var imgPath string
		var bytesPerChannel int
		imgPath, bytesPerChannel, width, height, err = b.readImage(suffix, importPath, item)
		if err != nil {
			return alignedImages, err
		}

		// Assign all images to PMC 1 onwards

		// TODO: might need non-square pixels with different x/y scales, or do something to align them if aspect ratio
		// between this image and spectrum differs. The one example GSQ file we have does this and it's not clear how
		// they should be aligned, so this is only a temporary scale factor for testing
		scale := float32(width) / float32(spectrumWidth) //float32(spectrumWidth) / float32(width)

		img := dataConvertModels.MatchedAlignedImageMeta{
			AlignedBeamPMC:       1,
			MatchedImageName:     scanId + "/" + filepath.Base(imgPath),
			MatchedImageFullPath: imgPath,
			XOffset:              0,
			YOffset:              0,
			XScale:               scale,
			YScale:               scale,
		}

		alignedImages = append(alignedImages, img)
		nextPMC++

		lastBytesPerChannels = append(lastBytesPerChannels, bytesPerChannel)
		lastWidths = append(lastWidths, width)
		lastHeights = append(lastHeights, height)
		lastSignals = append(lastSignals, signal)

		// At this point if we discover the last 3 can form an RGB image, make one!
		if len(lastWidths) > 2 {
			startIdx := len(lastWidths) - 3
			matches := true
			for c := 1; c < 3; c++ {
				if lastWidths[startIdx+c] != lastWidths[startIdx] ||
					lastHeights[startIdx+c] != lastHeights[startIdx] ||
					lastBytesPerChannels[startIdx+c] != lastBytesPerChannels[startIdx] ||
					lastSignals[startIdx+c] != lastSignals[startIdx] {
					matches = false
					break
				}
			}

			if matches {
				rgbPath, err := b.readAsRGBImage(suffix, importPath, images[itemIdx-2:itemIdx+1], width, height, 0, 0, 1, 1)
				if err != nil {
					return alignedImages, err
				}

				rgbImg := dataConvertModels.MatchedAlignedImageMeta{
					AlignedBeamPMC:       img.AlignedBeamPMC,
					MatchedImageName:     scanId + "/" + filepath.Base(rgbPath),
					MatchedImageFullPath: rgbPath,
					XOffset:              img.XOffset,
					YOffset:              img.YOffset,
					XScale:               img.XScale,
					YScale:               img.YScale,
				}
				alignedImages = append(alignedImages, rgbImg)
				nextPMC++
			}
		}
	}

	return alignedImages, nil
}

func (b Bruker) readAsRGBImage(suffix string,
	importPath string,
	items []*brukerreader.HspyItem,
	width, height int,
	xOffset, yOffset float64,
	xScale, yScale float64) (string, error) {
	if len(items) != 3 {
		return "", fmt.Errorf("readAsRGBImage with %v images", len(items))
	}

	sampleName := ""
	if f, err := getMetaField(items[0].Metadata, "Sample", "name"); err == nil {
		sampleName = f.(string)
	} else {
		return "", err
	}

	rgb := image.NewRGBA(image.Rect(0, 0, width, height))

	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			idx := (y*width + x)
			channels := []uint8{}
			for c := 0; c < len(items); c++ {
				item := items[c]

				if uint16data, ok := item.Data.([]uint16); ok {
					channels = append(channels, uint8(uint16data[idx]/256))
				} else if uint8data, ok := item.Data.([]uint8); ok {
					channels = append(channels, uint8data[idx])
				}
			}

			rgb.SetRGBA(x, y, color.RGBA{R: channels[0], G: channels[1], B: channels[2], A: 255})
		}
	}

	pngPath := filepath.Join(importPath, fmt.Sprintf("%v_rgb_%v.png", sampleName, suffix))
	if err := b.writeImage(pngPath, rgb); err != nil {
		return "", err
	}

	return pngPath, nil
}

func (b Bruker) writeImage(writeFilePath string, img image.Image) error {
	pngFile, err := os.Create(writeFilePath)
	if err != nil {
		return err
	}

	defer pngFile.Close()

	err = png.Encode(pngFile, img)
	if err != nil {
		return err
	}

	return nil
}
