package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func TestPaymentRepository(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_payment_repo.db")

	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open sqlite db: %v", err)
	}
	defer db.Close()

	// Seed patient first
	patientRepo := sqlite.NewPatientRepository(db)
	ctx := context.Background()
	patient := &domain.Patient{
		ID:        "pat_pay_1",
		FirstName: "Bob",
		LastName:  "Smith",
	}
	if err := patientRepo.Create(ctx, patient); err != nil {
		t.Fatalf("Failed to create patient: %v", err)
	}

	repo := sqlite.NewPaymentRepository(db)

	// 1. Input validation for Create
	invalidPayment := &domain.Payment{ID: "", PatientID: "pat_pay_1", Amount: 100, Date: "2026-08-15"}
	if err := repo.Create(ctx, invalidPayment); err == nil {
		t.Errorf("Expected error when creating payment with empty ID")
	}

	zeroAmountPayment := &domain.Payment{ID: "pay_0", PatientID: "pat_pay_1", Amount: 0, Date: "2026-08-15"}
	if err := repo.Create(ctx, zeroAmountPayment); err == nil {
		t.Errorf("Expected error when creating payment with Amount <= 0")
	}

	// 2. Create valid payment
	p1 := &domain.Payment{
		ID:        "pay_101",
		PatientID: "pat_pay_1",
		ClaimID:   "clm_1",
		Amount:    15000,
		Method:    domain.PaymentMethodCreditCard,
		Date:      "2026-08-15",
		Notes:     "Co-pay payment at front desk",
	}

	if err := repo.Create(ctx, p1); err != nil {
		t.Fatalf("Failed to create payment p1: %v", err)
	}

	p2 := &domain.Payment{
		ID:        "pay_102",
		PatientID: "pat_pay_1",
		Amount:    5000,
		Method:    domain.PaymentMethodCash,
		Date:      "2026-08-16",
	}

	if err := repo.Create(ctx, p2); err != nil {
		t.Fatalf("Failed to create payment p2: %v", err)
	}

	// 3. GetByID
	fetched, err := repo.GetByID(ctx, "pay_101")
	if err != nil {
		t.Fatalf("Failed to get payment by ID: %v", err)
	}
	if fetched.PatientID != "pat_pay_1" || fetched.Amount != 15000 || fetched.Method != domain.PaymentMethodCreditCard {
		t.Errorf("Unexpected payment data: %+v", fetched)
	}

	if _, err := repo.GetByID(ctx, ""); err == nil {
		t.Errorf("Expected error for empty ID in GetByID")
	}
	if _, err := repo.GetByID(ctx, "non_existent_pay"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound for non-existent payment, got: %v", err)
	}

	// 4. GetTotalPaid
	totalPaid, err := repo.GetTotalPaid(ctx, "pat_pay_1")
	if err != nil {
		t.Fatalf("Failed to get total paid: %v", err)
	}
	if totalPaid != 20000 {
		t.Errorf("Expected total paid 20000, got %d", totalPaid)
	}

	// Total paid for patient with no payments should be 0
	totalPaidUnk, err := repo.GetTotalPaid(ctx, "pat_no_payments")
	if err != nil {
		t.Fatalf("Failed to get total paid for patient without payments: %v", err)
	}
	if totalPaidUnk != 0 {
		t.Errorf("Expected total paid 0, got %d", totalPaidUnk)
	}

	// 5. List Payments
	listPatient, err := repo.List(ctx, "pat_pay_1")
	if err != nil {
		t.Fatalf("Failed to list payments by patient: %v", err)
	}
	if len(listPatient) != 2 {
		t.Errorf("Expected 2 payments for patient, got %d", len(listPatient))
	}

	listAll, err := repo.List(ctx, "")
	if err != nil {
		t.Fatalf("Failed to list all payments: %v", err)
	}
	if len(listAll) != 2 {
		t.Errorf("Expected 2 payments in total list, got %d", len(listAll))
	}

	// 6. Delete Payment
	if err := repo.Delete(ctx, "pay_101"); err != nil {
		t.Fatalf("Failed to delete payment: %v", err)
	}
	if _, err := repo.GetByID(ctx, "pay_101"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound after deletion, got: %v", err)
	}

	totalAfterDelete, err := repo.GetTotalPaid(ctx, "pat_pay_1")
	if err != nil {
		t.Fatalf("Failed to get total paid after delete: %v", err)
	}
	if totalAfterDelete != 5000 {
		t.Errorf("Expected total paid 5000 after deleting p1, got %d", totalAfterDelete)
	}

	if err := repo.Delete(ctx, "pay_101"); err != storage.ErrNotFound {
		t.Errorf("Expected ErrNotFound deleting already deleted payment, got: %v", err)
	}
}
