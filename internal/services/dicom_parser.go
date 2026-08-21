package services

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/png"

	"github.com/suyashkumar/dicom"
	"github.com/suyashkumar/dicom/pkg/tag"
)

// ParseDicomImages parses a DICOM file from bytes and extracts all image frames.
// It returns a slice of image.Image objects.
func ParseDicomImages(data []byte) ([]image.Image, error) {
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

	var images []image.Image
	for _, frame := range pixelDataInfo.Frames {
		// GetImage extracts the image.Image from the frame
		img, err := frame.GetImage()
		if err != nil {
			return nil, fmt.Errorf("failed to get image from frame: %w", err)
		}
		images = append(images, img)
	}

	if len(images) == 0 {
		return nil, errors.New("could not extract any images from frames")
	}

	return images, nil
}

// ImagesToBase64DataURLs converts a slice of image.Image to a slice of PNG data URLs.
func ImagesToBase64DataURLs(images []image.Image) ([]string, error) {
	var dataURLs []string

	for _, img := range images {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode image to PNG: %w", err)
		}

		b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
		dataURL := fmt.Sprintf("data:image/png;base64,%s", b64)
		dataURLs = append(dataURLs, dataURL)
	}

	return dataURLs, nil
}
