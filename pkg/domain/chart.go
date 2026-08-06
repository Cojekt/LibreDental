package domain

import (
	"fmt"
	"strconv"
	"time"
)

// ToothSurface represents tooth surfaces (Mesial, Distal, Occlusal, Incisal, Facial/Buccal, Lingual).
type ToothSurface string

const (
	SurfaceMesial   ToothSurface = "M"
	SurfaceDistal   ToothSurface = "D"
	SurfaceOcclusal ToothSurface = "O"
	SurfaceIncisal  ToothSurface = "I"
	SurfaceFacial   ToothSurface = "F"
	SurfaceLingual  ToothSurface = "L"
)

// ToothStatus represents conditions or work on a tooth (e.g. Existing, Treatment Planned, Completed).
type ToothStatus string

const (
	ToothStatusExisting         ToothStatus = "existing"
	ToothStatusTreatmentPlanned ToothStatus = "treatment_planned"
	ToothStatusCompleted        ToothStatus = "completed"
	ToothStatusMissing          ToothStatus = "missing"
)

// ToothCondition represents a specific tooth finding or procedure (e.g. Caries, Crown, Root Canal, Composite).
type ToothCondition struct {
	ID          string         `json:"id"`
	PatientID   string         `json:"patient_id"`
	ToothNumber int            `json:"tooth_number"` // Universal Numbering System 1-32 (or 101-120 for primary A-T)
	Surfaces    []ToothSurface `json:"surfaces,omitempty"`
	ADACode     string         `json:"ada_code,omitempty"` // e.g., D2392 for 2-surface posterior composite
	Description string         `json:"description"`
	Status      ToothStatus    `json:"status"`
	Fee         float64        `json:"fee,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// DentalChart represents a patient's full dental chart.
type DentalChart struct {
	PatientID  string           `json:"patient_id"`
	Conditions []ToothCondition `json:"conditions"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// FormatToothDisplay returns the localized display label for a tooth given a ToothSystem (Universal, FDI, Palmer).
func FormatToothDisplay(toothNumber int, system ToothSystem) string {
	// Adult teeth: 1..32
	if toothNumber >= 1 && toothNumber <= 32 {
		switch system {
		case ToothSystemFDI:
			return adultToFDI(toothNumber)
		case ToothSystemPalmer:
			return adultToPalmer(toothNumber)
		case ToothSystemUniversal:
			fallthrough
		default:
			return strconv.Itoa(toothNumber)
		}
	}

	// Primary teeth: 101..120 (A..T)
	if toothNumber >= 101 && toothNumber <= 120 {
		idx := toothNumber - 101 // 0..19
		switch system {
		case ToothSystemFDI:
			return primaryToFDI(idx)
		case ToothSystemPalmer:
			return primaryToPalmer(idx)
		case ToothSystemUniversal:
			fallthrough
		default:
			letters := []string{
				"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
				"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
			}
			if idx >= 0 && idx < len(letters) {
				return letters[idx]
			}
			return strconv.Itoa(toothNumber)
		}
	}

	return strconv.Itoa(toothNumber)
}

func adultToFDI(num int) string {
	// Maxillary Right 1..8 -> 18..11
	if num >= 1 && num <= 8 {
		return fmt.Sprintf("1%d", 9-num)
	}
	// Maxillary Left 9..16 -> 21..28
	if num >= 9 && num <= 16 {
		return fmt.Sprintf("2%d", num-8)
	}
	// Mandibular Left 17..24 -> 31..38
	if num >= 17 && num <= 24 {
		return fmt.Sprintf("3%d", num-16)
	}
	// Mandibular Right 25..32 -> 41..48
	if num >= 25 && num <= 32 {
		return fmt.Sprintf("4%d", 33-num)
	}
	return strconv.Itoa(num)
}

func adultToPalmer(num int) string {
	if num >= 1 && num <= 8 {
		return fmt.Sprintf("UR%d", 9-num)
	}
	if num >= 9 && num <= 16 {
		return fmt.Sprintf("UL%d", num-8)
	}
	if num >= 17 && num <= 24 {
		return fmt.Sprintf("LL%d", num-16)
	}
	if num >= 25 && num <= 32 {
		return fmt.Sprintf("LR%d", 33-num)
	}
	return strconv.Itoa(num)
}

func primaryToFDI(idx int) string {
	// Upper Right (A..E) -> 55..51
	if idx >= 0 && idx <= 4 {
		return fmt.Sprintf("5%d", 5-idx)
	}
	// Upper Left (F..J) -> 61..65
	if idx >= 5 && idx <= 9 {
		return fmt.Sprintf("6%d", idx-4)
	}
	// Lower Left (K..O) -> 71..75
	if idx >= 10 && idx <= 14 {
		return fmt.Sprintf("7%d", idx-9)
	}
	// Lower Right (P..T) -> 81..85
	if idx >= 15 && idx <= 19 {
		return fmt.Sprintf("8%d", 20-idx)
	}
	return strconv.Itoa(idx + 101)
}

func primaryToPalmer(idx int) string {
	letters := []string{"A", "B", "C", "D", "E"}
	if idx >= 0 && idx <= 4 {
		return "UR" + letters[4-idx]
	}
	if idx >= 5 && idx <= 9 {
		return "UL" + letters[idx-5]
	}
	if idx >= 10 && idx <= 14 {
		return "LL" + letters[idx-10]
	}
	if idx >= 15 && idx <= 19 {
		return "LR" + letters[19-idx]
	}
	return strconv.Itoa(idx + 101)
}
