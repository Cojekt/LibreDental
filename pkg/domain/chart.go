package domain

import (
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
	ToothNumber int            `json:"tooth_number"` // Universal Numbering System 1-32 (or A-T)
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
