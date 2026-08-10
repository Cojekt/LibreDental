package domain

import (
	"time"
)

// AppointmentStatus represents the current status of an appointment.
type AppointmentStatus string

const (
	AppointmentStatusScheduled AppointmentStatus = "scheduled"
	AppointmentStatusConfirmed AppointmentStatus = "confirmed"
	AppointmentStatusArrived   AppointmentStatus = "arrived"
	AppointmentStatusInChair   AppointmentStatus = "in_chair"
	AppointmentStatusComplete  AppointmentStatus = "completed"
	AppointmentStatusCancelled AppointmentStatus = "cancelled"
	AppointmentStatusNoShow    AppointmentStatus = "no_show"
)

// Appointment represents a scheduled dental visit.
type Appointment struct {
	ID          string            `json:"id"`
	PatientID   string            `json:"patient_id"`
	ProviderID  string            `json:"provider_id"`  // Dentist or Hygienist ID
	OperatoryID string            `json:"operatory_id"` // Chair/Room ID
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	Status      AppointmentStatus `json:"status"`
	Reason      string            `json:"reason"`
	Color       string            `json:"color,omitempty"`
	Notes       string            `json:"notes,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Version     int64             `json:"version"`
}

// AppointmentFilter specifies search parameters for appointments.
type AppointmentFilter struct {
	PatientID   string    `json:"patient_id,omitempty"`
	ProviderID  string    `json:"provider_id,omitempty"`
	OperatoryID string    `json:"operatory_id,omitempty"`
	StartDate   time.Time `json:"start_date,omitempty"`
	EndDate     time.Time `json:"end_date,omitempty"`
}
