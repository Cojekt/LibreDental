package services

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// ParseDicomDataURLs parses a DICOM file from bytes, extracts image frames (up to a limit),
// and encodes them directly to data URLs to limit memory usage.
func ParseDicomDataURLs(data []byte) ([]string, error) {
	// Parse the DICOM dataset from bytes
	dataset, err := dicom.Parse(bytes.NewReader(data), int64(len(data)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DICOM: %w", err)
	}

	// Find the PixelData element
	pixelDataElement, err := dataset.FindElementByTag(tag.PixelData)
	if err != nil {
		return nil, errors.New("pixel data element not found in DICOM dataset")
	}

	// Extract the PixelDataInfo
	pixelDataInfo := dicom.MustGetPixelDataInfo(pixelDataElement.Value)

	if pixelDataInfo.IntentionallySkipped {
		return nil, errors.New("pixel data was intentionally skipped")
	}

	if len(pixelDataInfo.Frames) == 0 {
		return nil, errors.New("no image frames found in DICOM pixel data")
	}

	var dataURLs []string
	const maxFrames = 50

	for i, frame := range pixelDataInfo.Frames {
		if i >= maxFrames {
			break
		}

		// Try to get the image
		img, err := frame.GetImage()
		if err == nil {
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err == nil {
				b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
				dataURLs = append(dataURLs, fmt.Sprintf("data:image/png;base64,%s", b64))
				continue
			}
		}

		// Fall back to encapsulated data
		if frame.IsEncapsulated() {
			encFrame, encErr := frame.GetEncapsulatedFrame()
			if encErr == nil && encFrame != nil && len(encFrame.Data) > 0 {
				b64 := base64.StdEncoding.EncodeToString(encFrame.Data)
				dataURLs = append(dataURLs, fmt.Sprintf("data:image/jpeg;base64,%s", b64))
				continue
			}
		}

		if len(dataURLs) == 0 {
			return nil, fmt.Errorf("failed to get image from frame: %w", err)
		}
	}

	if len(dataURLs) == 0 {
		return nil, errors.New("could not extract any images from frames")
	}

	return dataURLs, nil
}
