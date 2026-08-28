package domain

import (
	"time"
)

// TimeSlot represents a continuous block of operating hours.
type TimeSlot struct {
	OpenTime  string `json:"open_time"`  // e.g. "08:00"
	CloseTime string `json:"close_time"` // e.g. "17:00"
}

// ScheduleBreak represents a planned gap or closure during operating hours (e.g. Lunch Break, Staff Meeting).
type ScheduleBreak struct {
	Name      string `json:"name"`       // e.g. "Lunch Break", "Staff Meeting"
	StartTime string `json:"start_time"` // e.g. "12:00"
	EndTime   string `json:"end_time"`   // e.g. "13:00"
}

// BusinessHourDay represents daily practice operating hours.
type BusinessHourDay struct {
	Day       string          `json:"day"`        // e.g. "Monday", "Tuesday"
	OpenTime  string          `json:"open_time"`  // e.g. "08:00"
	CloseTime string          `json:"close_time"` // e.g. "17:00"
	IsClosed  bool            `json:"is_closed"`
	Slots     []TimeSlot      `json:"slots,omitempty"`
	Breaks    []ScheduleBreak `json:"breaks,omitempty"`
}

// DefaultBusinessHours returns a standard weekly schedule.
func DefaultBusinessHours() []BusinessHourDay {
	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	schedule := make([]BusinessHourDay, 7)
	for i, day := range days {
		isWeekend := (day == "Saturday" || day == "Sunday")
		breaks := []ScheduleBreak{}
		if !isWeekend {
			breaks = append(breaks, ScheduleBreak{
				Name:      "Lunch Break",
				StartTime: "12:00",
				EndTime:   "13:00",
			})
		}
		schedule[i] = BusinessHourDay{
			Day:       day,
			OpenTime:  "08:00",
			CloseTime: "17:00",
			IsClosed:  isWeekend,
			Slots: []TimeSlot{
				{OpenTime: "08:00", CloseTime: "17:00"},
			},
			Breaks: breaks,
		}
	}
	return schedule
}

type ProviderRole string

const (
	RoleDentist   ProviderRole = "dentist"
	RoleHygienist ProviderRole = "hygienist"
	RoleAssistant ProviderRole = "assistant"
	RoleStaff     ProviderRole = "staff"
)

type Provider struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Role          ProviderRole `json:"role"`
	Specialty     string       `json:"specialty"`
	LicenseNumber string       `json:"license_number"`
	Email         string       `json:"email"`
	Phone         string       `json:"phone"`
	Color         string       `json:"color"`
	Pin           string       `json:"pin"`
	IsActive      bool         `json:"is_active"`
	HourlyRate    int64        `json:"hourly_rate"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

type OperatoryType string

const (
	OperatoryTypeGeneral OperatoryType = "general"
	OperatoryTypeHygiene OperatoryType = "hygiene"
	OperatoryTypeSurgery OperatoryType = "surgery"
)

type Operatory struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	RoomCode  string        `json:"room_code"`
	Type      OperatoryType `json:"type"`
	IsActive  bool          `json:"is_active"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// PracticeConfig represents system-wide clinic profile and regional configuration.
type PracticeConfig struct {
	ID            int64             `json:"id"`
	ClinicName    string            `json:"clinic_name"`
	Tagline       string            `json:"tagline"`
	TaxID         string            `json:"tax_id"`
	LicenseNumber string            `json:"license_number"`
	Phone         string            `json:"phone"`
	Email         string            `json:"email"`
	Website       string            `json:"website"`
	AddressLine1  string            `json:"address_line1"`
	AddressLine2  string            `json:"address_line2"`
	City          string            `json:"city"`
	StateProvince string            `json:"state_province"`
	PostalCode    string            `json:"postal_code"`
	CountryCode   CountryCode       `json:"country_code"`
	Currency      string            `json:"currency"`
	ToothSystem   ToothSystem       `json:"tooth_system"`
	DateFormat    string            `json:"date_format"`
	BusinessHours []BusinessHourDay `json:"business_hours"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// NewPracticeConfig creates a new PracticeConfig populated with regional defaults from country metadata.
func NewPracticeConfig(meta CountryConfig) *PracticeConfig {
	now := time.Now().UTC()

	return &PracticeConfig{
		ID:            1,
		ClinicName:    "My Dental Clinic",
		CountryCode:   meta.Code,
		Currency:      meta.DefaultCurrency,
		ToothSystem:   meta.DefaultToothSystem,
		DateFormat:    meta.DateFormat,
		BusinessHours: DefaultBusinessHours(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
