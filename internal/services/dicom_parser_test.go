package services

import (
	"testing"
)

func TestParseDicomDataURLs_InvalidInput(t *testing.T) {
	t.Run("EmptyBytes", func(t *testing.T) {
		_, err := ParseDicomDataURLs([]byte{})
		if err == nil {
			t.Error("Expected error for empty input, got nil")
		}
	})

	t.Run("RandomBytes", func(t *testing.T) {
		_, err := ParseDicomDataURLs([]byte("this is not a dicom file at all"))
		if err == nil {
			t.Error("Expected error for non-DICOM input, got nil")
		}
	})

	t.Run("NilBytes", func(t *testing.T) {
		_, err := ParseDicomDataURLs(nil)
		if err == nil {
			t.Error("Expected error for nil input, got nil")
		}
	})

	t.Run("TruncatedDICOMHeader", func(t *testing.T) {
		// DICOM files start with 128 bytes of preamble followed by "DICM" magic.
		// A file with just the magic but no valid dataset should fail to parse.
		header := make([]byte, 132)
		copy(header[128:], "DICM")
		_, err := ParseDicomDataURLs(header)
		if err == nil {
			t.Error("Expected error for truncated DICOM header, got nil")
		}
	})
}

// NOTE: Positive-path tests (valid DICOM files with pixel data) require real
// DICOM fixture files. Add them to a testdata/ directory and extend this suite
// when fixtures are available.
