package demo

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// archivedPatientCount patients have left the practice (moved, transferred care, deceased).
const archivedPatientCount = 15

// seedRemovals exercises every removal path the app offers, through the same services and
// so with the same audit entries a real office would leave behind: mistaken records that
// were deleted, patients archived, a departed hygienist deactivated, a room taken down.
func seedRemovals(g *ServiceGraph, token string, rng *rand.Rand, h *historyState, docIDs []string, now time.Time, summary *SeedSummary) error {
	deleted := func(err error, what string) error {
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", what, err)
		}
		summary.DeletedRecords++
		return nil
	}

	// Appointments booked in error: double bookings and wrong-patient entries, created and
	// then removed. Upcoming ones sit in the next-year window, past ones in the history.
	for i := range 10 {
		pid := h.adults.next(h, "")
		day := h.today.AddDate(0, 0, -3-rng.IntN(7*historyWeeks-5))
		if i%2 == 0 {
			day = h.today.AddDate(1, 0, 3+rng.IntN(7*futureWeeks))
		}
		for day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			day = day.AddDate(0, 0, 1)
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), 8+rng.IntN(8), 0, 0, 0, h.loc)
		if start.Hour() == 12 {
			start = start.Add(time.Hour)
		}
		a := &domain.Appointment{
			ID: h.id("appt"), PatientID: pid, ProviderID: "prov_101", OperatoryID: "op_1",
			StartTime: start, EndTime: start.Add(30 * time.Minute), Status: domain.AppointmentStatusScheduled,
			Reason: "Duplicate booking", Color: "#3b82f6", Notes: "Booked twice from the online request form.",
			CreatedAt: now, UpdatedAt: now, Version: 1,
		}
		if _, err := g.Appointment.CreateAppointment(token, a); err != nil {
			return fmt.Errorf("failed to seed appointment %s: %w", a.ID, err)
		}
		if err := deleted(g.Appointment.DeleteAppointment(token, a.ID), a.ID); err != nil {
			return err
		}
	}

	// Chart entries recorded on the wrong tooth, and planned treatment the patient declined.
	for i := range 6 {
		pid := h.adults.next(h, "")
		p := h.filling(pid)
		if p.tooth == 0 {
			continue
		}
		status := domain.ToothStatusExisting
		if i >= 3 {
			status = domain.ToothStatusTreatmentPlanned
		}
		c := &domain.SaveToothConditionPayload{
			ID: h.id("cond"), PatientID: pid, ToothNumber: p.tooth, Surfaces: p.surfaces,
			ADACode: p.code, Description: catalogDesc[p.code], Status: status,
		}
		if status == domain.ToothStatusTreatmentPlanned {
			src, err := h.price(pid, "prov_101", p.code)
			if err != nil {
				return err
			}
			c.Fee, c.Pricing = src.Amount, src
		}
		if _, err := g.Chart.SaveToothCondition(token, c); err != nil {
			return fmt.Errorf("failed to seed tooth condition %s: %w", c.ID, err)
		}
		if err := deleted(g.Chart.DeleteToothCondition(token, c.ID), c.ID); err != nil {
			return err
		}
	}

	// Duplicate claims: a second draft opened for a visit that was already billed.
	for _, orig := range pickN(rng, h.claims, 4) {
		dup := &domain.Claim{
			ID: h.id("claim"), PatientID: orig.PatientID, ProviderID: orig.ProviderID,
			DateOfService: orig.DateOfService, Status: domain.ClaimStatusDraft,
			Notes: "Duplicate of " + orig.ID + "; opened by mistake.",
		}
		src, err := h.price(orig.PatientID, orig.ProviderID, "D0120")
		if err != nil {
			return err
		}
		dup.LineItems = []domain.ClaimLineItem{{ID: h.id("li"), ADACode: "D0120", Description: catalogDesc["D0120"], Fee: src.Amount, Pricing: src, Surfaces: []domain.ToothSurface{}}}
		if _, err := g.Billing.CreateClaim(token, dup); err != nil {
			return fmt.Errorf("failed to seed claim %s: %w", dup.ID, err)
		}
		if err := deleted(g.Billing.DeleteClaim(token, dup.ID), dup.ID); err != nil {
			return err
		}
	}

	// Payments keyed in twice at checkout, caught and removed at end-of-day reconciliation.
	for _, orig := range pickN(rng, h.payments, 5) {
		dup := orig
		dup.ID, dup.Notes = h.id("pay"), "Duplicate entry"
		if _, err := g.Billing.RecordPayment(token, &dup); err != nil {
			return fmt.Errorf("failed to seed payment %s: %w", dup.ID, err)
		}
		if err := deleted(g.Billing.DeletePayment(token, dup.ID), dup.ID); err != nil {
			return err
		}
	}

	// Documents uploaded to the wrong patient's chart.
	for _, id := range pickN(rng, docIDs, 3) {
		if err := deleted(g.Document.DeleteDocument(token, id), id); err != nil {
			return err
		}
		summary.DocumentsCount--
	}

	// Billing setup that was tried and withdrawn.
	promo := &domain.TreatmentBundle{
		ID: "bundle_6", Shortname: "whitening-promo", Name: "Spring Whitening Promotion",
		Description: "Discontinued seasonal offer",
		Items:       []domain.BundleItemTemplate{{ADACode: "D1110", Description: "Prophylaxis - Adult", DefaultFee: 9900}},
		TotalFee:    9900, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := g.Billing.CreateBundle(token, promo); err != nil {
		return fmt.Errorf("failed to seed treatment bundle %s: %w", promo.ID, err)
	}
	if err := deleted(g.Billing.DeleteBundle(token, promo.ID), promo.ID); err != nil {
		return err
	}
	fs := &domain.FeeSchedule{ID: "fee_sched_6", CountryCode: domain.CountryUS, Code: "D1206", CustomFee: 2500, UpdatedAt: now}
	if _, err := g.Billing.SaveFeeSchedule(token, fs); err != nil {
		return fmt.Errorf("failed to seed fee schedule %s: %w", fs.ID, err)
	}
	if err := deleted(g.Billing.DeleteFeeSchedule(token, fs.ID), fs.ID); err != nil {
		return err
	}
	guardian := h.plans["Guardian Dental"]
	if err := g.Insurance.SaveFee(token, &domain.InsuranceFee{ProviderID: guardian, CountryCode: domain.CountryUS, Code: "D7210", Fee: 24000}); err != nil {
		return fmt.Errorf("failed to seed Guardian D7210 fee: %w", err)
	}
	if err := deleted(g.Insurance.DeleteFee(token, guardian, string(domain.CountryUS), "D7210"), "Guardian D7210 fee"); err != nil {
		return err
	}

	// A temporary chair that was never used, and a hygienist who has left the practice. Both
	// are deactivated rather than erased, so they stay on file.
	if err := g.Practice.DeleteOperatory(token, "op_10"); err != nil {
		return fmt.Errorf("failed to deactivate op_10: %w", err)
	}
	if err := g.Practice.DeleteProvider(token, "prov_110"); err != nil {
		return fmt.Errorf("failed to deactivate prov_110: %w", err)
	}

	// Shifts entered against a Saturday the office was closed, then removed.
	saturday := h.today.AddDate(0, 0, -9)
	stamp := time.Date(saturday.Year(), saturday.Month(), saturday.Day(), 15, 0, 0, 0, time.UTC)
	for _, id := range []string{"prov_108", "prov_111"} {
		if err := g.Timecard.CreateManualTimecard(token, id, 480, stamp.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("failed to seed timecard for %s: %w", id, err)
		}
		cards, err := g.Timecard.ListTimecards(id, stamp.Format(time.RFC3339), stamp.Format(time.RFC3339))
		if err != nil || len(cards) != 1 {
			return fmt.Errorf("failed to find mistaken timecard for %s: %v", id, err)
		}
		if err := deleted(g.Timecard.DeleteTimecard(token, cards[0].ID), cards[0].ID); err != nil {
			return err
		}
	}

	// Patients who opted out of automated reminders after joining.
	for range 5 {
		p, err := g.Patient.GetPatient(token, h.adults.next(h, ""))
		if err != nil {
			return err
		}
		p.ReminderOptIn = false
		p.UpdatedAt = now
		if _, err := g.Patient.UpdatePatient(token, p); err != nil {
			return fmt.Errorf("failed to update patient %s: %w", p.ID, err)
		}
	}

	// Archive patients who have no upcoming bookings.
	archived := 0
	for _, pid := range h.adults.ids {
		if archived == archivedPatientCount {
			break
		}
		if h.future[pid] || !chance(rng, 0.15) {
			continue
		}
		if err := g.Patient.ArchivePatient(token, pid); err != nil {
			return fmt.Errorf("failed to archive patient %s: %w", pid, err)
		}
		archived++
		summary.ArchivedPatients++
	}
	return nil
}

func pickN[T any](rng *rand.Rand, xs []T, n int) []T {
	out := make([]T, 0, n)
	for _, i := range rng.Perm(len(xs))[:min(n, len(xs))] {
		out = append(out, xs[i])
	}
	return out
}
