package demo

import (
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/services"
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

// SeedDatabase populates a fresh save folder with sample practice configuration,
// healthcare providers, operatory treatment rooms, patient records, scheduled
// appointments, dental charting conditions, billing records, and documents.
//
// Every record is created by calling the same Go services the Wails frontend binds
// to (internal/services), not by writing to the database directly, so the generated
// demo data can never drift from real application behavior.
func SeedDatabase(db *sqlite.DB, auditDb *sqlite.DB, appDir, demoDataDir string) (*SeedSummary, error) {
	summary := &SeedSummary{}
	graph := BuildServiceGraph(db, auditDb, appDir)

	// Fixed reference time to ensure deterministic demo database generation.
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	today := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	// 1. Practice config
	if err := seedPracticeConfig(graph, now, summary); err != nil {
		return nil, err
	}

	// 2. Bootstrap: create the very first provider (unauthenticated, exactly as a
	// fresh install's onboarding flow would) and immediately use it to open a
	// session. Every subsequent step authenticates with that session token, the
	// same way a logged-in staff member's requests would.
	token, err := bootstrapFirstProvider(graph, now, summary)
	if err != nil {
		return nil, err
	}

	// 3. Remaining providers
	if err := seedProviders(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 4. Operatories
	if err := seedOperatories(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 5. Patients
	if err := seedPatients(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 6. Appointments
	if err := seedAppointments(graph, token, now, today, summary); err != nil {
		return nil, err
	}

	// 7. Dental Chart Conditions
	if err := seedChartConditions(graph, token, summary); err != nil {
		return nil, err
	}

	// 8. Custom Fee Schedules
	if err := seedFeeSchedules(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 9. Treatment Bundles
	if err := seedBundles(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 10. Claims
	if err := seedClaims(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 11. Payments
	if err := seedPayments(graph, token, now, summary); err != nil {
		return nil, err
	}

	// 12. Documents
	if err := seedDocuments(graph, token, demoDataDir, summary); err != nil {
		return nil, err
	}

	// 13. Local app settings (config.json), so the save folder is a complete,
	// ready-to-use drop-in replacement for a real appDir, not just its databases.
	if err := seedSystemSettings(appDir); err != nil {
		return nil, err
	}

	return summary, nil
}

func seedSystemSettings(appDir string) error {
	settings := services.NewSystemSettingsService(appDir)
	if err := settings.SetTheme("system"); err != nil {
		return fmt.Errorf("failed to seed app settings: %w", err)
	}
	return nil
}
