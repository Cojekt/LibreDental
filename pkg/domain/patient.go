package domain

import (
	"time"
)

// Gender represents the patient's gender identity.
type Gender string

const (
	GenderMale      Gender = "male"
	GenderFemale    Gender = "female"
	GenderOther     Gender = "other"
	GenderUndisclosed Gender = "undisclosed"
)

// Status represents the patient's record status.
type Status string

const (
	StatusActive   Status = "active"
	StatusArchived Status = "archived"
)

// Patient represents a dental patient demographic record.
type Patient struct {
	ID             string    `json:"id"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	MiddleName     string    `json:"middle_name,omitempty"`
	PreferredName  string    `json:"preferred_name,omitempty"`
	DateOfBirth    time.Time `json:"date_of_birth"`
	Gender         Gender    `json:"gender"`
	Email          string    `json:"email,omitempty"`
	PhonePrimary   string    `json:"phone_primary,omitempty"`
	PhoneSecondary string    `json:"phone_secondary,omitempty"`
	AddressLine1   string    `json:"address_line1,omitempty"`
	AddressLine2   string    `json:"address_line2,omitempty"`
	City           string    `json:"city,omitempty"`
	State          string    `json:"state,omitempty"`
	ZipCode        string    `json:"zip_code,omitempty"`
	MedicalAlerts  []string  `json:"medical_alerts,omitempty"`
	Allergies      []string  `json:"allergies,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Version        int64     `json:"version"`
	Status         Status    `json:"status"`
}

// PatientFilter specifies search/filter parameters for patients.
type PatientFilter struct {
	Query  string `json:"query,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}
