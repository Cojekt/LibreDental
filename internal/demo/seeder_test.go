package demo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

type volumeRange struct{ min, max int }

// Generated volumes per size: the small set is only as big as coverage needs (see
// testCoverage), the large one looks like a busy multi-chair practice without becoming a
// stress test.
var expectedVolumes = map[Size]map[string]volumeRange{
	SizeSmall: {
		"appointments": {30, 100}, "conditions": {30, 120}, "claims": {15, 60},
		"payments": {10, 50}, "documents": {15, 35}, "timecards": {15, 40}, "deleted records": {8, 20},
	},
	SizeLarge: {
		"appointments": {1000, 2000}, "conditions": {1500, 3500}, "claims": {700, 1400},
		"payments": {500, 1200}, "documents": {100, 200}, "timecards": {120, 250}, "deleted records": {25, 60},
	},
}

func TestSeedDatabase(t *testing.T) {
	for _, size := range Sizes {
		t.Run(string(size), func(t *testing.T) { testSeedDatabase(t, size) })
	}
}

func testSeedDatabase(t *testing.T, size Size) {
	prof := profiles[size]
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_demo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test sqlite db: %v", err)
	}
	defer db.Close()

	auditDbPath := filepath.Join(tempDir, "test_demo_audit.db")
	auditDb, err := sqlite.OpenAudit(auditDbPath)
	if err != nil {
		t.Fatalf("Failed to open test audit sqlite db: %v", err)
	}
	defer auditDb.Close()

	appDir := t.TempDir()
	summary, err := SeedDatabase(db, auditDb, appDir, "./data", size)
	if err != nil {
		t.Fatalf("SeedDatabase failed: %v", err)
	}

	if !summary.PracticeConfigured {
		t.Errorf("Expected PracticeConfigured to be true")
	}
	exact := []struct {
		name      string
		got, want int
	}{
		{"providers", summary.ProvidersCount, 11},
		{"operatories", summary.OperatoriesCount, 10},
		{"patients", summary.PatientsCount, 6 + prof.patients},
		{"archived patients", summary.ArchivedPatients, prof.archivedPatients},
		{"fee schedule overrides", summary.FeeSchedulesCount, 5},
		{"treatment bundles", summary.BundlesCount, 5},
		{"insurance plans", summary.InsurancePlans, 8},
		{"insurance plan fees", summary.InsuranceFees, 44},
	}
	for _, c := range exact {
		if c.got != c.want {
			t.Errorf("Expected %d %s, got %d", c.want, c.name, c.got)
		}
	}
	got := map[string]int{
		"appointments": summary.AppointmentsCount, "conditions": summary.ConditionsCount,
		"claims": summary.ClaimsCount, "payments": summary.PaymentsCount,
		"documents": summary.DocumentsCount, "timecards": summary.TimecardsCount,
		"deleted records": summary.DeletedRecords,
	}
	for name, want := range expectedVolumes[size] {
		if got[name] < want.min || got[name] > want.max {
			t.Errorf("Expected %d-%d %s, got %d", want.min, want.max, name, got[name])
		}
	}

	ctx := context.Background()
	testCoverage(t, db, ctx)

	// Verify practice config
	configRepo := sqlite.NewPracticeConfigRepository(db)
	cfg, err := configRepo.Get(ctx)
	if err != nil {
		t.Fatalf("Failed to query practice config: %v", err)
	}
	if cfg.ClinicName != "Apex Dental Studio" {
		t.Errorf("Expected ClinicName 'Apex Dental Studio', got '%s'", cfg.ClinicName)
	}

	// Verify patients
	patientRepo := sqlite.NewPatientRepository(db)
	patients, total, err := patientRepo.List(ctx, domain.PatientFilter{Status: string(domain.StatusArchived)})
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if total != int64(prof.archivedPatients) || len(patients) != prof.archivedPatients {
		t.Errorf("Expected %d archived patients in list, got total %d, count %d", prof.archivedPatients, total, len(patients))
	}
	archived := map[string]bool{}
	for _, p := range patients {
		archived[p.ID] = true
	}

	// Verify appointments
	apptRepo := sqlite.NewAppointmentRepository(db)
	appts, err := apptRepo.List(ctx, domain.AppointmentFilter{})
	if err != nil {
		t.Fatalf("Failed to list appointments: %v", err)
	}
	if len(appts) != summary.AppointmentsCount {
		t.Errorf("Expected %d appointments in list, got %d", summary.AppointmentsCount, len(appts))
	}

	laLoc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		laLoc = time.FixedZone("PDT", -7*3600)
	}

	uniqueDays := make(map[string]bool)
	afterHoursCount := 0

	for _, a := range appts {
		if archived[a.PatientID] && (a.Status == domain.AppointmentStatusScheduled || a.Status == domain.AppointmentStatusConfirmed) {
			t.Errorf("Archived patient %s still has upcoming appointment %s", a.PatientID, a.ID)
		}
		st := a.StartTime.In(laLoc)
		et := a.EndTime.In(laLoc)

		dayStr := st.Format("2006-01-02")
		uniqueDays[dayStr] = true

		startHour := st.Hour()
		endHour := et.Hour()
		endMin := et.Minute()

		// Normal business hours: 08:00 - 17:00
		isOutsideHours := startHour < 8 || startHour >= 17 || (endHour > 17 || (endHour == 17 && endMin > 0))
		if isOutsideHours {
			afterHoursCount++
		}
	}

	if len(uniqueDays) <= 1 {
		t.Errorf("Expected appointments to be spread across multiple days, got %d unique day(s)", len(uniqueDays))
	}

	if afterHoursCount != 1 {
		t.Errorf("Expected exactly 1 appointment outside normal business hours, got %d", afterHoursCount)
	}

	// Verify dental chart
	chartRepo := sqlite.NewChartRepository(db)
	chart, err := chartRepo.GetChart(ctx, "pat_101")
	if err != nil {
		t.Fatalf("Failed to get chart for pat_101: %v", err)
	}
	if len(chart.Conditions) != 2 {
		t.Errorf("Expected 2 conditions for pat_101, got %d", len(chart.Conditions))
	}

	// Every billable treatment carries a confirmed pricing stamp matching its saved fee.
	wantPricing := map[string]struct {
		kind, clinician string
		amount          int64
	}{
		"cond_301": {"standard", "prov_101", 29500}, // clinician fee schedule
		"cond_303": {"standard", "prov_101", 23000}, // catalog fee
		"cond_304": {"standard", "prov_101", 17000},
		"cond_305": {"insurance", "prov_102", 88000}, // MetLife plan fee
		"cond_306": {"standard", "prov_102", 130000}, // practice-wide fee schedule
		"cond_308": {"manual", "", 140000},
		"cond_312": {"standard", "prov_103", 5500},
	}
	for id, want := range wantPricing {
		c, err := chartRepo.GetConditionByID(ctx, id)
		if err != nil {
			t.Fatalf("Failed to get condition %s: %v", id, err)
		}
		p := c.Pricing
		if p == nil || p.ConfirmedAt.IsZero() || p.Kind != want.kind || p.ClinicianID != want.clinician || p.Amount != want.amount || c.Fee != want.amount {
			t.Errorf("Condition %s: fee %d, pricing %+v; want %+v", id, c.Fee, p, want)
		}
	}
	for _, id := range []string{"cond_302", "cond_307"} {
		if c, err := chartRepo.GetConditionByID(ctx, id); err != nil || c.Pricing != nil {
			t.Errorf("Expected non-billable condition %s to have no pricing (err %v)", id, err)
		}
	}

	claimRepo := sqlite.NewClaimRepository(db)
	claims, err := claimRepo.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list claims: %v", err)
	}
	if len(claims) != summary.ClaimsCount {
		t.Errorf("Expected %d claims in list, got %d", summary.ClaimsCount, len(claims))
	}
	for _, claim := range claims {
		for _, li := range claim.LineItems {
			if li.Pricing == nil || li.Pricing.Amount != li.Fee || li.Pricing.ClinicianID != claim.ProviderID {
				t.Errorf("Claim %s line %s: fee %d, pricing %+v", claim.ID, li.ID, li.Fee, li.Pricing)
			}
		}
	}

	// Verify documents
	docRepo := sqlite.NewDocumentRepository(db)
	clinicDocs, err := docRepo.ListClinicDocuments()
	if err != nil {
		t.Fatalf("Failed to list clinic documents: %v", err)
	}
	if len(clinicDocs) != 4 {
		t.Errorf("Expected 4 clinic documents, got %d", len(clinicDocs))
	}

	pat101ID := "pat_101"
	patDocs, err := docRepo.ListByFilter(domain.DocumentFilter{PatientID: &pat101ID})
	if err != nil {
		t.Fatalf("Failed to list patient documents: %v", err)
	}
	if len(patDocs) != 7 {
		t.Errorf("Expected 7 patient documents for pat_101, got %d", len(patDocs))
	}

	for _, id := range []string{"pat_101", "pat_102", "pat_103", "pat_104", "pat_105", "pat_106"} {
		docs, err := docRepo.ListByFilter(domain.DocumentFilter{PatientID: &id})
		if err != nil {
			t.Fatalf("Failed to list documents for %s: %v", id, err)
		}
		if len(docs) == 0 {
			t.Errorf("Expected demo documents for %s, got none", id)
		}
	}

	// Verify the save folder is a complete drop-in appDir: app settings were written,
	// and the audit trail recorded real activity from driving the services.
	if _, err := os.Stat(filepath.Join(appDir, "config.json")); err != nil {
		t.Errorf("Expected config.json to be written to appDir: %v", err)
	}

	auditRepo := sqlite.NewAuditRepository(auditDb)
	auditEntries, err := auditRepo.Query(ctx, "", 1, "")
	if err != nil {
		t.Fatalf("Failed to query audit log: %v", err)
	}
	if len(auditEntries) == 0 {
		t.Errorf("Expected at least one audit log entry from seeding through services")
	}

	// Removed records are gone from the main database but stay in the audit trail.
	var deletes int
	if err := auditDb.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_logs WHERE action = 'DELETE'`).Scan(&deletes); err != nil {
		t.Fatalf("Failed to count audited deletions: %v", err)
	}
	if deletes < summary.DeletedRecords {
		t.Errorf("Expected at least %d audited deletions, got %d", summary.DeletedRecords, deletes)
	}
	providers, err := configRepo.ListProviders(ctx)
	if err != nil {
		t.Fatalf("Failed to list providers: %v", err)
	}
	for _, p := range providers {
		if p.IsActive == (p.ID == "prov_110") {
			t.Errorf("Provider %s: unexpected active state %v", p.ID, p.IsActive)
		}
	}
	ops, err := configRepo.ListOperatories(ctx)
	if err != nil {
		t.Fatalf("Failed to list operatories: %v", err)
	}
	for _, op := range ops {
		if op.IsActive == (op.ID == "op_9" || op.ID == "op_10") {
			t.Errorf("Operatory %s: unexpected active state %v", op.ID, op.IsActive)
		}
	}
}

// testCoverage checks that every status, method and scheduled provider the UI can show has at
// least one record, so a smaller profile can't silently drop a case.
func testCoverage(t *testing.T, db *sqlite.DB, ctx context.Context) {
	t.Helper()
	distinct := func(query string) map[string]bool {
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		defer rows.Close()
		out := map[string]bool{}
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				t.Fatalf("%s: %v", query, err)
			}
			out[v] = true
		}
		return out
	}
	want := func(what, query string, values ...string) {
		got := distinct(query)
		for _, v := range values {
			if !got[v] {
				t.Errorf("Expected at least one %s %q", what, v)
			}
		}
	}
	str := func(xs ...any) []string {
		out := make([]string, len(xs))
		for i, x := range xs {
			out[i] = fmt.Sprint(x)
		}
		return out
	}

	want("appointment status", `SELECT DISTINCT status FROM appointments`, str(
		domain.AppointmentStatusScheduled, domain.AppointmentStatusConfirmed, domain.AppointmentStatusArrived,
		domain.AppointmentStatusInChair, domain.AppointmentStatusComplete, domain.AppointmentStatusCancelled,
		domain.AppointmentStatusNoShow)...)
	want("claim status", `SELECT DISTINCT status FROM claims`, str(
		domain.ClaimStatusDraft, domain.ClaimStatusSubmitted, domain.ClaimStatusAccepted,
		domain.ClaimStatusRejected, domain.ClaimStatusPaid)...)
	want("payment method", `SELECT DISTINCT method FROM payments`, str(
		domain.PaymentMethodCash, domain.PaymentMethodCheck, domain.PaymentMethodCreditCard,
		domain.PaymentMethodInsurance, domain.PaymentMethodWriteOff)...)
	want("tooth condition status", `SELECT DISTINCT status FROM dental_conditions`, str(
		domain.ToothStatusExisting, domain.ToothStatusTreatmentPlanned, domain.ToothStatusCompleted,
		domain.ToothStatusMissing)...)

	var providers []string
	for _, col := range scheduleColumns() {
		providers = append(providers, col.provider)
	}
	want("past visit for provider", `SELECT DISTINCT provider_id FROM appointments WHERE status IN ('completed', 'cancelled', 'no_show')`, providers...)
	want("timecard payroll state", `SELECT DISTINCT CASE WHEN paid_at IS NULL THEN 'owed' ELSE 'paid' END FROM timecards`, "owed", "paid")
}
