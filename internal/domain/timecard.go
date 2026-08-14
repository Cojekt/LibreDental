package domain

import "time"

type Timecard struct {
	ID         string     `json:"id"`
	ProviderID string     `json:"provider_id"`
	ClockIn    time.Time  `json:"clock_in"`
	ClockOut   *time.Time `json:"clock_out"`
	HourlyRate float64    `json:"hourly_rate"`
	TotalHours float64    `json:"total_hours"`
	TotalPay   float64    `json:"total_pay"`
	PaidAt     *time.Time `json:"paid_at"`
	IsManual   bool       `json:"is_manual"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
