package sqlite_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestPatientRepository_CRUD(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_libredental.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPatientRepository(db)
	ctx := context.Background()

	// 1. Create Patient
	patient := &domain.Patient{
		ID:                              "pat_123",
		FirstName:                       "John",
		LastName:                        "Doe",
		DateOfBirth:                     time.Date(1985, 5, 20, 0, 0, 0, 0, time.UTC),
		Sex:                             domain.SexMale,
		Email:                           "john.doe@example.com",
		PhonePrimary:                    "555-0199",
		EmergencyContactName:            "Mary Doe",
		EmergencyContactRel:             "Spouse",
		EmergencyContactPhone:           "555-0188",
		InsuranceCarrier:                "Delta Dental",
		InsurancePolicyNumber:           "DEL-12345",
		InsurancePayerID:                "CDCA1",
		InsuranceSubscriberFirstName:    "Mary",
		InsuranceSubscriberLastName:     "Doe",
		InsuranceSubscriberDOB:          "1983-02-01",
		InsuranceSubscriberSex:          domain.SexFemale,
		InsuranceSubscriberRelationship: domain.SubscriberRelationshipSpouse,
		InsuranceSubscriberCity:         "Buckeye",
		InsuranceSubscriberPostalCode:   "85326",
		StateProvince:                   "CA",
		PostalCode:                      "90210",
		CountryCode:                     domain.CountryUS,
		NationalIDType:                  "ssn",
		NationalID:                      "123-45-6789",
		MedicalAlerts:                   []string{"Penicillin Allergy", "High Blood Pressure"},
		Allergies:                       []string{"Latex"},
	}

	err = repo.Create(ctx, patient)
	if err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	// 2. Read Patient
	fetched, err := repo.GetByID(ctx, "pat_123")
	if err != nil {
		t.Fatalf("Failed to get patient by ID: %v", err)
	}
	if fetched.FirstName != "John" || fetched.LastName != "Doe" {
		t.Errorf("Unexpected patient name: %s %s", fetched.FirstName, fetched.LastName)
	}
	if fetched.NationalID != "123-45-6789" || fetched.CountryCode != domain.CountryUS {
		t.Errorf("Unexpected national ID / country: %s / %s", fetched.NationalID, fetched.CountryCode)
	}
	if len(fetched.MedicalAlerts) != 2 {
		t.Errorf("Expected 2 medical alerts, got %d", len(fetched.MedicalAlerts))
	}
	if fetched.InsurancePayerID != "CDCA1" || fetched.InsuranceSubscriberFirstName != "Mary" ||
		fetched.InsuranceSubscriberLastName != "Doe" || fetched.InsuranceSubscriberDOB != "1983-02-01" ||
		fetched.InsuranceSubscriberRelationship != domain.SubscriberRelationshipSpouse ||
		fetched.InsuranceSubscriberSex != domain.SexFemale || fetched.InsuranceSubscriberCity != "Buckeye" ||
		fetched.InsuranceSubscriberPostalCode != "85326" {
		t.Errorf("Unexpected payer/policyholder fields: %+v", fetched)
	}

	// 3. Update Patient (Optimistic Concurrency)
	fetched.LastName = "Smith"
	err = repo.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Failed to update patient: %v", err)
	}
	if fetched.Version != 2 {
		t.Errorf("Expected version 2, got %d", fetched.Version)
	}

	// Attempt stale update (should cause storage.ErrConflict)
	stalePatient := &domain.Patient{
		ID:        "pat_123",
		FirstName: "Stale",
		LastName:  "Stale",
		Version:   1, // Stale version
	}
	err = repo.Update(ctx, stalePatient)
	if err != storage.ErrConflict {
		t.Errorf("Expected ErrConflict for stale update, got: %v", err)
	}

	// 4. List Patients
	patients, total, err := repo.List(ctx, domain.PatientFilter{Query: "smith"})
	if err != nil {
		t.Fatalf("Failed to list patients: %v", err)
	}
	if total != 1 || len(patients) != 1 {
		t.Errorf("Expected 1 result for 'smith', got total=%d len=%d", total, len(patients))
	}

	// 5. Delete Patient
	err = repo.Delete(ctx, "pat_123")
	if err != nil {
		t.Fatalf("Failed to delete patient: %v", err)
	}

	_, err = repo.GetByID(ctx, "pat_123")
	if err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}
}

func TestPatientRepository_ListReturnsAllWithoutLimit(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test_list_all.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPatientRepository(db)
	ctx := context.Background()

	const n = 60
	for i := range n {
		p := &domain.Patient{
			ID:          fmt.Sprintf("pat_%03d", i),
			FirstName:   "Test",
			LastName:    fmt.Sprintf("Patient%03d", i),
			DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			Sex:         domain.SexOther,
			Status:      domain.StatusActive,
		}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Failed to create patient %d: %v", i, err)
		}
	}

	all, total, err := repo.List(ctx, domain.PatientFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != n || len(all) != n {
		t.Errorf("Expected all %d patients without a limit, got %d (total %d)", n, len(all), total)
	}

	page, _, err := repo.List(ctx, domain.PatientFilter{Limit: 10, Offset: 55})
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}
	if len(page) != 5 {
		t.Errorf("Expected explicit limit/offset to still apply (5 results), got %d", len(page))
	}
}

// Dates are stored as YYYY-MM-DD; the subscriber DOB is optional, so it may also be blank.
func TestPatientRepository_RejectsNonCanonicalSubscriberDOB(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "test_libredental.db"))
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	repo := sqlite.NewPatientRepository(db)
	ctx := context.Background()
	for i, tc := range []struct {
		dob string
		ok  bool
	}{
		{"", true},
		{"1983-02-01", true},
		{"02/01/1983", false},
		{"1983-02-01T00:00:00Z", false},
		{"1983-02-30", false},
	} {
		err := repo.Create(ctx, &domain.Patient{
			ID:                     fmt.Sprintf("pat_sub_%d", i),
			FirstName:              "Ann",
			LastName:               "Lee",
			DateOfBirth:            time.Date(2010, 3, 4, 0, 0, 0, 0, time.UTC),
			Sex:                    domain.SexFemale,
			InsuranceSubscriberDOB: tc.dob,
		})
		if tc.ok && err != nil {
			t.Errorf("subscriber DOB %q rejected: %v", tc.dob, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("subscriber DOB %q accepted", tc.dob)
		}
	}
}

func TestPatientRepository_Search(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	repo := sqlite.NewPatientRepository(db)
	ctx := context.Background()

	seed := []struct{ id, first, last, email string }{
		{"p1", "Ágnes", "Őri", "agnes@example.com"},
		{"p2", "John", "Smith", "john.smith@example.com"},
		{"p3", "Zoé", "Müller", "zoe@example.com"},
	}
	for _, s := range seed {
		if err := repo.Create(ctx, &domain.Patient{
			ID: s.id, FirstName: s.first, LastName: s.last, Email: s.email,
			DateOfBirth: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), Sex: domain.SexFemale,
		}); err != nil {
			t.Fatalf("create %s: %v", s.id, err)
		}
	}

	only := func(t *testing.T, query, wantID string) {
		t.Helper()
		got, _, err := repo.List(ctx, domain.PatientFilter{Query: query})
		if err != nil {
			t.Fatalf("list %q: %v", query, err)
		}
		if len(got) != 1 || got[0].ID != wantID {
			ids := make([]string, len(got))
			for i, p := range got {
				ids[i] = p.ID
			}
			t.Errorf("search %q = %v, want exactly [%s]", query, ids, wantID)
		}
	}

	// Accent-insensitive case folding: SQLite's built-in LOWER() only folds ASCII,
	// so a lowercase accented query never matched a capitalised accented name.
	only(t, "ágnes", "p1")
	only(t, "ŐRI", "p1")
	only(t, "müller", "p3")

	// Multi-word queries should match across first and last name in either order.
	only(t, "John Smith", "p2")
	only(t, "Smith John", "p2")

	// LIKE wildcards typed by the user are data, not patterns: "%" must not list everyone.
	if got, _, err := repo.List(ctx, domain.PatientFilter{Query: "%"}); err != nil {
		t.Fatalf("wildcard list: %v", err)
	} else if len(got) != 0 {
		t.Errorf("search %q returned %d patients, want 0 (literal match)", "%", len(got))
	}
}
