package services

import (
	"errors"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/LibreDental/libredental/internal/domain"
)

// ErrDocumentTypeNotOpenable is returned for file types the OS might run rather than display.
var ErrDocumentTypeNotOpenable = errors.New("this file type cannot be opened from LibreDental")

// openableExtensions are the only file types handed to the OS default application. It is an
// allow-list because the ways a file can run code (programs, scripts, shortcuts, installers,
// macro documents) differ between systems and can't be listed reliably.
var openableExtensions = map[string]bool{
	".pdf": true, ".txt": true, ".csv": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true,
	".tif": true, ".tiff": true, ".dcm": true, ".dicom": true,
	".docx": true, ".xlsx": true, ".odt": true, ".ods": true,
}

// extensionsByContentType names the extension for a document whose name lacks a usable one.
// The system MIME table is not used: it differs between machines.
var extensionsByContentType = map[string]string{
	"application/pdf":   ".pdf",
	"text/plain":        ".txt",
	"text/csv":          ".csv",
	"image/png":         ".png",
	"image/jpeg":        ".jpg",
	"image/gif":         ".gif",
	"image/webp":        ".webp",
	"image/bmp":         ".bmp",
	"image/tiff":        ".tif",
	"application/dicom": ".dcm",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       ".xlsx",
	"application/vnd.oasis.opendocument.text":                                 ".odt",
	"application/vnd.oasis.opendocument.spreadsheet":                          ".ods",
}

// maxOpenNameBytes keeps the temp file name, with its prefix and extension, within the
// 255-byte limit of common file systems.
const maxOpenNameBytes = 150

// openableFileName returns a safe file name for a temp copy of doc that will be opened. The
// result always ends in an allowed extension. A name ending in anything else gets the content
// type's extension appended, so the OS treats it as that type; without one it is refused.
func openableFileName(doc *domain.Document) (string, error) {
	name := filepath.Base(doc.Name)
	ext := filepath.Ext(name)
	if openableExtensions[strings.ToLower(ext)] {
		name = strings.TrimSuffix(name, ext)
		ext = strings.ToLower(ext)
	} else {
		contentType, _, _ := strings.Cut(doc.ContentType, ";")
		ext = extensionsByContentType[strings.ToLower(strings.TrimSpace(contentType))]
		if ext == "" {
			return "", ErrDocumentTypeNotOpenable
		}
	}

	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsDigit(r) || strings.ContainsRune(" ._-", r) {
			return r
		}
		return '_'
	}, name)
	for len(name) > maxOpenNameBytes {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	if strings.Trim(name, " .") == "" {
		name = "document"
	}
	return name + ext, nil
}
