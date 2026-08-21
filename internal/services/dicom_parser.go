package services

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"

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

		var frameErr error

		// Try to get the image
		img, err := frame.GetImage()
		if err == nil {
			var buf bytes.Buffer

			// Normalize Gray16 to Gray 8-bit to fix darkness and improve performance
			if gray16, ok := img.(*image.Gray16); ok {
				bounds := gray16.Bounds()
				gray8 := image.NewGray(bounds)

				var minVal uint16 = 65535
				var maxVal uint16 = 0
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						c := gray16.Gray16At(x, y)
						if c.Y < minVal {
							minVal = c.Y
						}
						if c.Y > maxVal {
							maxVal = c.Y
						}
					}
				}

				if maxVal == minVal {
					maxVal = minVal + 1
				}

				scale := 255.0 / float64(maxVal-minVal)
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						c := gray16.Gray16At(x, y)
						val8 := uint8(float64(c.Y-minVal) * scale)
						gray8.SetGray(x, y, color.Gray{Y: val8})
					}
				}

				if encodeErr := jpeg.Encode(&buf, gray8, &jpeg.Options{Quality: 90}); encodeErr == nil {
					b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
					dataURLs = append(dataURLs, fmt.Sprintf("data:image/jpeg;base64,%s", b64))
					continue
				} else {
					frameErr = fmt.Errorf("failed to encode normalized frame to jpeg: %w", encodeErr)
				}
			} else {
				if encodeErr := png.Encode(&buf, img); encodeErr == nil {
					b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
					dataURLs = append(dataURLs, fmt.Sprintf("data:image/png;base64,%s", b64))
					continue
				} else {
					frameErr = fmt.Errorf("failed to encode frame image to png: %w", encodeErr)
				}
			}
		} else {
			frameErr = fmt.Errorf("failed to get image from frame: %w", err)
		}

		// Fall back to encapsulated data
		if frame.IsEncapsulated() {
			encFrame, encErr := frame.GetEncapsulatedFrame()
			if encErr == nil && encFrame != nil && len(encFrame.Data) > 0 {
				mimeType := http.DetectContentType(encFrame.Data)
				if mimeType == "application/octet-stream" {
					if len(encFrame.Data) >= 12 && string(encFrame.Data[4:8]) == "jP  " {
						mimeType = "image/jp2"
					} else if len(encFrame.Data) >= 2 && encFrame.Data[0] == 0xFF && encFrame.Data[1] == 0x4F {
						mimeType = "image/jp2"
					} else if len(encFrame.Data) >= 2 && encFrame.Data[0] == 0xFF && encFrame.Data[1] == 0xD8 {
						mimeType = "image/jpeg"
					} else {
						mimeType = "image/jpeg" // Final fallback
					}
				}

				b64 := base64.StdEncoding.EncodeToString(encFrame.Data)
				dataURLs = append(dataURLs, fmt.Sprintf("data:%s;base64,%s", mimeType, b64))
				continue
			} else if encErr != nil {
				frameErr = fmt.Errorf("failed to get encapsulated frame: %w", encErr)
			} else {
				frameErr = errors.New("encapsulated frame was empty")
			}
		}

		return nil, frameErr
	}

	if len(dataURLs) == 0 {
		return nil, errors.New("could not extract any images from frames")
	}

	return dataURLs, nil
}
