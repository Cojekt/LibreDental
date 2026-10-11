package services

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/LibreDental/libredental/internal/storage"
)

// SMS encodings. One character outside the GSM 03.38 alphabet, such as a curly apostrophe
// pasted from a word processor, switches the whole message to UCS-2 and cuts each part from
// 160 characters to 70. Every part is billed.
const (
	smsEncodingGSM7 = "GSM-7"
	smsEncodingUCS2 = "UCS-2"
)

// Per-part and per-message limits from AWS End User Messaging SMS. Multipart messages lose
// a few characters per part to the header that reassembles them.
const (
	smsGSM7SinglePart = 160
	smsGSM7MultiPart  = 153
	smsGSM7Max        = 1530
	smsUCS2SinglePart = 70
	smsUCS2MultiPart  = 67
	smsUCS2Max        = 630
)

const (
	smsGSM7Basic    = "@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà"
	smsGSM7Extended = "^{}\\[]~|€\f"
)

// smsSize reports the encoding a message will be sent with, its length in that encoding's
// units (GSM-7 extended characters count twice; UCS-2 counts UTF-16 code units), and the
// number of billed parts.
func smsSize(body string) (encoding string, units, parts int) {
	encoding = smsEncodingGSM7
	for _, r := range body {
		switch {
		case strings.ContainsRune(smsGSM7Basic, r):
			units++
		case strings.ContainsRune(smsGSM7Extended, r):
			units += 2
		default:
			encoding = smsEncodingUCS2
		}
	}

	single, multi := smsGSM7SinglePart, smsGSM7MultiPart
	if encoding == smsEncodingUCS2 {
		units = len(utf16.Encode([]rune(body)))
		single, multi = smsUCS2SinglePart, smsUCS2MultiPart
	}
	switch {
	case units == 0:
		parts = 0
	case units <= single:
		parts = 1
	default:
		parts = (units + multi - 1) / multi
	}
	return encoding, units, parts
}

// checkSMSBody rejects text messages that are empty or longer than AWS accepts, before any
// request is made.
func checkSMSBody(body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("%w: text message body is empty", storage.ErrInvalidInput)
	}
	encoding, units, _ := smsSize(body)
	limit := smsGSM7Max
	if encoding == smsEncodingUCS2 {
		limit = smsUCS2Max
	}
	if units > limit {
		return fmt.Errorf("%w: text message is %d characters in %s, over the %d-character limit",
			storage.ErrInvalidInput, units, encoding, limit)
	}
	return nil
}
