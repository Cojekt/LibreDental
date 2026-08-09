package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func seedAppointments(ctx context.Context, appointmentRepo *sqlite.AppointmentRepository, now time.Time, today time.Time, summary *SeedSummary) error {
	twoDaysAgo := today.AddDate(0, 0, -2)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	inTwoDays := today.AddDate(0, 0, 2)

	laLoc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		laLoc = time.FixedZone("PDT", -7*3600)
	}

	mkTime := func(day time.Time, hour int, minute int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, laLoc)
	}

	appointments := []*domain.Appointment{
		{
			ID:          "appt_201",
			PatientID:   "pat_101",
			ProviderID:  "prov_101",
			OperatoryID: "op_1",
			StartTime:   mkTime(twoDaysAgo, 9, 0),
			EndTime:     mkTime(twoDaysAgo, 10, 0),
			Status:      domain.AppointmentStatusComplete,
			Reason:      "Annual Comprehensive Examination & X-Rays",
			Color:       "#3b82f6",
			Notes:       "Exam complete. Advised composite filling for Tooth #3.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_202",
			PatientID:   "pat_102",
			ProviderID:  "prov_103",
			OperatoryID: "op_2",
			StartTime:   mkTime(yesterday, 10, 0),
			EndTime:     mkTime(yesterday, 11, 0),
			Status:      domain.AppointmentStatusComplete,
			Reason:      "Biannual Prophylaxis & Fluoride Treatment",
			Color:       "#f59e0b",
			Notes:       "Periodontal charting stable. Recommended electric toothbrush.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_203",
			PatientID:   "pat_103",
			ProviderID:  "prov_102",
			OperatoryID: "op_3",
			StartTime:   mkTime(yesterday, 14, 0),
			EndTime:     mkTime(yesterday, 15, 0),
			Status:      domain.AppointmentStatusComplete,
			Reason:      "Emergency Evaluation #19 Toothache",
			Color:       "#10b981",
			Notes:       "Diagnosis: Irreversible pulpitis #19. Treatment plan: Root Canal.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_204",
			PatientID:   "pat_101",
			ProviderID:  "prov_101",
			OperatoryID: "op_1",
			StartTime:   mkTime(today, 9, 0),
			EndTime:     mkTime(today, 10, 30),
			Status:      domain.AppointmentStatusInChair,
			Reason:      "Composite Restoration #3 (MO)",
			Color:       "#3b82f6",
			Notes:       "Local anesthesia administered. Caries removed cleanly.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_205",
			PatientID:   "pat_104",
			ProviderID:  "prov_103",
			OperatoryID: "op_2",
			StartTime:   mkTime(today, 11, 0),
			EndTime:     mkTime(today, 12, 0),
			Status:      domain.AppointmentStatusArrived,
			Reason:      "Routine Cleaning & Hygiene Instruction",
			Color:       "#f59e0b",
			Notes:       "Patient checked in at reception.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_206",
			PatientID:   "pat_105",
			ProviderID:  "prov_101",
			OperatoryID: "op_1",
			StartTime:   mkTime(today, 14, 0),
			EndTime:     mkTime(today, 15, 0),
			Status:      domain.AppointmentStatusConfirmed,
			Reason:      "Crown Consultation #14",
			Color:       "#3b82f6",
			Notes:       "Confirmed via SMS notification.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_207",
			PatientID:   "pat_106",
			ProviderID:  "prov_102",
			OperatoryID: "op_3",
			StartTime:   mkTime(today, 18, 0),
			EndTime:     mkTime(today, 19, 0),
			Status:      domain.AppointmentStatusScheduled,
			Reason:      "After-Hours Teeth Whitening Consultation",
			Color:       "#10b981",
			Notes:       "After-hours appointment per patient request. In-office vs take-home tray comparison.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_208",
			PatientID:   "pat_102",
			ProviderID:  "prov_101",
			OperatoryID: "op_1",
			StartTime:   mkTime(tomorrow, 9, 0),
			EndTime:     mkTime(tomorrow, 10, 0),
			Status:      domain.AppointmentStatusScheduled,
			Reason:      "Follow-up Composite Filling #30",
			Color:       "#3b82f6",
			Notes:       "Scheduled per treatment plan.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_209",
			PatientID:   "pat_103",
			ProviderID:  "prov_102",
			OperatoryID: "op_3",
			StartTime:   mkTime(tomorrow, 10, 30),
			EndTime:     mkTime(tomorrow, 12, 0),
			Status:      domain.AppointmentStatusScheduled,
			Reason:      "Root Canal Therapy #19 (Stage 1)",
			Color:       "#10b981",
			Notes:       "Pulpectomy & canal instrumentation.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
		{
			ID:          "appt_210",
			PatientID:   "pat_105",
			ProviderID:  "prov_103",
			OperatoryID: "op_2",
			StartTime:   mkTime(inTwoDays, 13, 0),
			EndTime:     mkTime(inTwoDays, 14, 0),
			Status:      domain.AppointmentStatusScheduled,
			Reason:      "Periodontal Maintenance Cleaning",
			Color:       "#f59e0b",
			Notes:       "3-month recall visit.",
			CreatedAt:   now,
			UpdatedAt:   now,
			Version:     1,
		},
	}

	for _, appt := range appointments {
		if err := appointmentRepo.Create(ctx, appt); err != nil {
			return fmt.Errorf("failed to seed appointment %s: %w", appt.ID, err)
		}
		summary.AppointmentsCount++
	}
	return nil
}
