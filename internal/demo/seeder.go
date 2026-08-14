package demo

import (
	"context"
	"time"

	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

// SeedSummary contains statistical metrics of the data populated into the database.
type SeedSummary struct {
	PracticeConfigured bool
	ProvidersCount     int
	OperatoriesCount   int
	PatientsCount      int
	AppointmentsCount  int
	ConditionsCount    int
	FeeSchedulesCount  int
	BundlesCount       int
	ClaimsCount        int
	PaymentsCount      int
	DocumentsCount     int
}

// SeedDatabase populates the target SQLite database with sample practice configuration,
// healthcare providers, operatory treatment rooms, patient records, scheduled appointments,
// and dental charting conditions.
func SeedDatabase(db *sqlite.DB, appDir, demoDataDir string) (*SeedSummary, error) {
	ctx := context.Background()
	summary := &SeedSummary{}

	patientRepo := sqlite.NewPatientRepository(db)
	appointmentRepo := sqlite.NewAppointmentRepository(db)
	practiceConfigRepo := sqlite.NewPracticeConfigRepository(db)
	chartRepo := sqlite.NewChartRepository(db)
	procedureRepo := sqlite.NewProcedureRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	claimRepo := sqlite.NewClaimRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)

	// Fixed reference time to ensure deterministic demo database generation
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// 1. Practice Config
	if err := seedPracticeConfig(ctx, practiceConfigRepo, now, summary); err != nil {
		return nil, err
	}

	// 2. Providers
	if err := seedProviders(ctx, practiceConfigRepo, now, summary); err != nil {
		return nil, err
	}

	// 3. Operatories
	if err := seedOperatories(ctx, practiceConfigRepo, now, summary); err != nil {
		return nil, err
	}

	// 4. Patients
	if err := seedPatients(ctx, patientRepo, now, summary); err != nil {
		return nil, err
	}

	// 5. Appointments
	if err := seedAppointments(ctx, appointmentRepo, now, today, summary); err != nil {
		return nil, err
	}

	// 6. Dental Chart Conditions
	if err := seedChartConditions(ctx, chartRepo, now, summary); err != nil {
		return nil, err
	}

	// 7. Custom Fee Schedules
	if err := seedFeeSchedules(ctx, procedureRepo, now, summary); err != nil {
		return nil, err
	}

	// 8. Treatment Bundles
	if err := seedBundles(ctx, bundleRepo, now, summary); err != nil {
		return nil, err
	}

	// 9. Claims
	if err := seedClaims(ctx, claimRepo, now, summary); err != nil {
		return nil, err
	}

	// 10. Payments
	if err := seedPayments(ctx, paymentRepo, now, summary); err != nil {
		return nil, err
	}

	// 11. Documents
	if err := seedDocuments(ctx, db, appDir, demoDataDir, now, summary); err != nil {
		return nil, err
	}

	return summary, nil
}
