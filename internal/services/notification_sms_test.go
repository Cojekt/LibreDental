package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/LibreDental/libredental/internal/storage"
)

func TestSMSSize(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		encoding string
		units    int
		parts    int
	}{
		{"empty", "", smsEncodingGSM7, 0, 0},
		{"one GSM part", strings.Repeat("a", 160), smsEncodingGSM7, 160, 1},
		{"two GSM parts", strings.Repeat("a", 161), smsEncodingGSM7, 161, 2},
		{"GSM part boundary", strings.Repeat("a", 306), smsEncodingGSM7, 306, 2},
		{"three GSM parts", strings.Repeat("a", 307), smsEncodingGSM7, 307, 3},
		{"accented GSM letters", "Café à Zürich", smsEncodingGSM7, 13, 1},
		{"accent outside GSM switches to UCS-2", "Ñandú", smsEncodingUCS2, 5, 1},
		{"extended characters count twice", "{€}", smsEncodingGSM7, 6, 1},
		{"extended pushes over one part", strings.Repeat("a", 159) + "€", smsEncodingGSM7, 161, 2},
		{"curly apostrophe switches to UCS-2", "Dr. Lee’s office", smsEncodingUCS2, 16, 1},
		{"one UCS-2 part", strings.Repeat("’", 70), smsEncodingUCS2, 70, 1},
		{"two UCS-2 parts", strings.Repeat("’", 71), smsEncodingUCS2, 71, 2},
		{"emoji is two UTF-16 units", "Hi 😀", smsEncodingUCS2, 5, 1},
		{"newlines are GSM", "Line 1\nLine 2\r\n", smsEncodingGSM7, 15, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoding, units, parts := smsSize(tt.body)
			if encoding != tt.encoding || units != tt.units || parts != tt.parts {
				t.Errorf("smsSize(%q) = %s, %d units, %d parts; want %s, %d, %d",
					tt.body, encoding, units, parts, tt.encoding, tt.units, tt.parts)
			}
		})
	}
}

func TestCheckSMSBody(t *testing.T) {
	for _, body := range []string{strings.Repeat("a", 1530), strings.Repeat("’", 630), "Reminder: appointment tomorrow at 2:00 PM."} {
		if err := checkSMSBody(body); err != nil {
			t.Errorf("Expected %d-character body to be accepted, got %v", len([]rune(body)), err)
		}
	}
	for _, body := range []string{"", "   \n", strings.Repeat("a", 1531), strings.Repeat("’", 631), strings.Repeat("€", 766)} {
		if err := checkSMSBody(body); !errors.Is(err, storage.ErrInvalidInput) {
			t.Errorf("Expected %d-character body to be rejected, got %v", len([]rune(body)), err)
		}
	}
}
