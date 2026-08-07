package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/pkg/domain"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
)

// SeedSummary contains statistical metrics of the data populated into the database.
type SeedSummary struct {
	PracticeConfigured bool
	ProvidersCount     int
	OperatoriesCount   int
	PatientsCount      int
	AppointmentsCount  int
	ConditionsCount    int
}

// SeedDatabase populates the target SQLite database with sample practice configuration,
// healthcare providers, operatory treatment rooms, patient records, scheduled appointments,
// and dental charting conditions.
func SeedDatabase(db *sqlite.DB) (*SeedSummary, error) {
	ctx := context.Background()
	summary := &SeedSummary{}

	patientRepo := sqlite.NewPatientRepository(db)
	appointmentRepo := sqlite.NewAppointmentRepository(db)
	practiceConfigRepo := sqlite.NewPracticeConfigRepository(db)
	chartRepo := sqlite.NewChartRepository(db)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	twoDaysAgo := today.AddDate(0, 0, -2)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)
	inTwoDays := today.AddDate(0, 0, 2)

	// 1. Practice Config
	config := &domain.PracticeConfig{
		ID:            1,
		ClinicName:    "Apex Dental Studio",
		Tagline:       "Modern Dental Care & Implant Center",
		TaxID:         "94-1234567",
		LicenseNumber: "DEN-CA-884920",
		Phone:         "(555) 234-5678",
		Email:         "info@apexdentalstudio.com",
		Website:       "https://apexdentalstudio.example.com",
		AddressLine1:  "101 Dental Plaza, Suite 200",
		City:          "San Francisco",
		StateProvince: "CA",
		PostalCode:    "94105",
		CountryCode:   domain.CountryUS,
		Currency:      "USD",
		ToothSystem:   domain.ToothSystemUniversal,
		DateFormat:    "MM/DD/YYYY",
		BusinessHours: domain.DefaultBusinessHours(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := practiceConfigRepo.Save(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to seed practice config: %w", err)
	}
	summary.PracticeConfigured = true

	// 2. Providers
	providers := []*domain.Provider{
		{
			ID:            "prov_101",
			Name:          "Dr. Sarah Jenkins",
			Role:          domain.RoleDentist,
			Specialty:     "General Dentistry & Restorative",
			LicenseNumber: "DEN-98212",
			Email:         "s.jenkins@apexdentalstudio.com",
			Phone:         "555-0101",
			Color:         "#3b82f6",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "prov_102",
			Name:          "Dr. Marcus Vance",
			Role:          domain.RoleDentist,
			Specialty:     "Endodontics & Surgery",
			LicenseNumber: "DEN-77419",
			Email:         "m.vance@apexdentalstudio.com",
			Phone:         "555-0102",
			Color:         "#10b981",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "prov_103",
			Name:          "Elena Rostova",
			Role:          domain.RoleHygienist,
			Specialty:     "Preventive Dental Hygiene",
			LicenseNumber: "HYG-33104",
			Email:         "e.rostova@apexdentalstudio.com",
			Phone:         "555-0103",
			Color:         "#f59e0b",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}

	for _, p := range providers {
		if err := practiceConfigRepo.SaveProvider(ctx, p); err != nil {
			return nil, fmt.Errorf("failed to seed provider %s: %w", p.Name, err)
		}
		summary.ProvidersCount++
	}

	// 3. Operatories
	operatories := []*domain.Operatory{
		{
			ID:        "op_1",
			Name:      "Operatory 1",
			RoomCode:  "Room 101",
			Type:      domain.OperatoryTypeGeneral,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_2",
			Name:      "Hygiene Suite",
			RoomCode:  "Room 102",
			Type:      domain.OperatoryTypeHygiene,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_3",
			Name:      "Surgical Suite",
			RoomCode:  "Room 103",
			Type:      domain.OperatoryTypeSurgery,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, op := range operatories {
		if err := practiceConfigRepo.SaveOperatory(ctx, op); err != nil {
			return nil, fmt.Errorf("failed to seed operatory %s: %w", op.Name, err)
		}
		summary.OperatoriesCount++
	}

	// 4. Patients
	patients := []*domain.Patient{
		{
			ID:             "pat_101",
			FirstName:      "Johnathan",
			LastName:       "Miller",
			DateOfBirth:    time.Date(1985, 4, 12, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderMale,
			Email:          "j.miller@example.com",
			PhonePrimary:   "(555) 111-2233",
			AddressLine1:   "742 Evergreen Terrace",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94107",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "123-45-6789",
			MedicalAlerts:  []string{"High Blood Pressure", "Latex Allergy"},
			Allergies:      []string{"Latex"},
			Notes:          "Prefers morning appointments.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
		{
			ID:             "pat_102",
			FirstName:      "Sophia",
			LastName:       "Martinez",
			DateOfBirth:    time.Date(1992, 9, 24, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderFemale,
			Email:          "sophia.m@example.com",
			PhonePrimary:   "(555) 222-3344",
			AddressLine1:   "1280 Mission Street, Apt 4B",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94103",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "987-65-4321",
			MedicalAlerts:  []string{"Penicillin Allergy", "Asthma"},
			Allergies:      []string{"Penicillin"},
			Notes:          "Pre-medicate with non-penicillin antibiotic before invasive procedures.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
		{
			ID:             "pat_103",
			FirstName:      "Robert",
			LastName:       "Chen",
			DateOfBirth:    time.Date(1968, 11, 5, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderMale,
			Email:          "r.chen@example.com",
			PhonePrimary:   "(555) 333-4455",
			AddressLine1:   "450 Sutter Street",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94108",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "456-78-9012",
			MedicalAlerts:  []string{"Type II Diabetes"},
			Allergies:      []string{},
			Notes:          "Monitor blood sugar level before surgical visits.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
		{
			ID:             "pat_104",
			FirstName:      "Emma",
			LastName:       "Watson",
			DateOfBirth:    time.Date(2001, 3, 18, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderFemale,
			Email:          "ewatson@example.com",
			PhonePrimary:   "(555) 444-5566",
			AddressLine1:   "88 King Street",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94107",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "321-65-9874",
			MedicalAlerts:  []string{},
			Allergies:      []string{},
			Notes:          "Clean medical record.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
		{
			ID:             "pat_105",
			FirstName:      "David",
			LastName:       "O'Connor",
			DateOfBirth:    time.Date(1955, 7, 30, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderMale,
			Email:          "doconnor@example.com",
			PhonePrimary:   "(555) 555-6677",
			AddressLine1:   "300 California Street",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94104",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "654-98-3210",
			MedicalAlerts:  []string{"Anticoagulant Therapy (Warfarin)"},
			Allergies:      []string{},
			Notes:          "Check INR value prior to extractions.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
		{
			ID:             "pat_106",
			FirstName:      "Mia",
			LastName:       "Tanaka",
			DateOfBirth:    time.Date(1998, 12, 14, 0, 0, 0, 0, time.UTC),
			Gender:         domain.GenderFemale,
			Email:          "mia.t@example.com",
			PhonePrimary:   "(555) 666-7788",
			AddressLine1:   "1500 Van Ness Avenue",
			City:           "San Francisco",
			StateProvince:  "CA",
			PostalCode:     "94109",
			CountryCode:    domain.CountryUS,
			NationalIDType: "ssn",
			NationalID:     "789-12-3456",
			MedicalAlerts:  []string{"Nitrous Oxide Sensitivity"},
			Allergies:      []string{},
			Notes:          "Interested in cosmetic whitening.",
			Status:         domain.StatusActive,
			CreatedAt:      now,
			UpdatedAt:      now,
			Version:        1,
		},
	}

	for _, p := range patients {
		if err := patientRepo.Create(ctx, p); err != nil {
			return nil, fmt.Errorf("failed to seed patient %s: %w", p.FirstName, err)
		}
		summary.PatientsCount++
	}

	// 5. Appointments
	mkTime := func(day time.Time, hour int, minute int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, time.UTC)
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
			return nil, fmt.Errorf("failed to seed appointment %s: %w", appt.ID, err)
		}
		summary.AppointmentsCount++
	}

	// 6. Dental Chart Conditions
	conditions := []*domain.ToothCondition{
		{
			ID:          "cond_301",
			PatientID:   "pat_101",
			ToothNumber: 3,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
			ADACode:     "D2392",
			Description: "2-Surface Posterior Composite Restoration",
			Status:      domain.ToothStatusCompleted,
			Fee:         280.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_302",
			PatientID:   "pat_101",
			ToothNumber: 14,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D2391",
			Description: "1-Surface Posterior Composite",
			Status:      domain.ToothStatusExisting,
			Fee:         190.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_303",
			PatientID:   "pat_102",
			ToothNumber: 30,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal, domain.SurfaceDistal},
			ADACode:     "D2393",
			Description: "3-Surface Posterior Composite",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         350.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_304",
			PatientID:   "pat_102",
			ToothNumber: 1,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D7140",
			Description: "Impacted Wisdom Tooth Extraction",
			Status:      domain.ToothStatusCompleted,
			Fee:         450.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_305",
			PatientID:   "pat_103",
			ToothNumber: 19,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D3330",
			Description: "Molar Endodontic Therapy (Root Canal)",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         1200.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_306",
			PatientID:   "pat_103",
			ToothNumber: 19,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D2740",
			Description: "Porcelain/Ceramic Crown",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         1350.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_307",
			PatientID:   "pat_103",
			ToothNumber: 32,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D7140",
			Description: "Missing Tooth (Extracted)",
			Status:      domain.ToothStatusMissing,
			Fee:         0.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_308",
			PatientID:   "pat_105",
			ToothNumber: 14,
			Surfaces:    []domain.ToothSurface{},
			ADACode:     "D2750",
			Description: "Porcelain Fused to High Noble Metal Crown",
			Status:      domain.ToothStatusTreatmentPlanned,
			Fee:         1400.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_309",
			PatientID:   "pat_105",
			ToothNumber: 18,
			Surfaces:    []domain.ToothSurface{domain.SurfaceMesial, domain.SurfaceOcclusal},
			ADACode:     "D2150",
			Description: "Amalgam Restoration 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         220.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_310",
			PatientID:   "pat_106",
			ToothNumber: 8,
			Surfaces:    []domain.ToothSurface{domain.SurfaceIncisal, domain.SurfaceFacial},
			ADACode:     "D2331",
			Description: "Anterior Composite 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         260.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_311",
			PatientID:   "pat_106",
			ToothNumber: 9,
			Surfaces:    []domain.ToothSurface{domain.SurfaceIncisal, domain.SurfaceFacial},
			ADACode:     "D2331",
			Description: "Anterior Composite 2 Surfaces",
			Status:      domain.ToothStatusExisting,
			Fee:         260.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "cond_312",
			PatientID:   "pat_104",
			ToothNumber: 30,
			Surfaces:    []domain.ToothSurface{domain.SurfaceOcclusal},
			ADACode:     "D1351",
			Description: "Dental Sealant - Per Tooth",
			Status:      domain.ToothStatusCompleted,
			Fee:         65.00,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	for _, cond := range conditions {
		if err := chartRepo.SaveCondition(ctx, cond); err != nil {
			return nil, fmt.Errorf("failed to seed tooth condition %s: %w", cond.ID, err)
		}
		summary.ConditionsCount++
	}

	return summary, nil
}
