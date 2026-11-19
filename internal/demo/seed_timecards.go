package demo

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"
)

// hourlyStaff clock time through the timecard screen; dentists are paid on production.
var hourlyStaff = []struct {
	id    string
	days  []time.Weekday
	until time.Time
}{
	{"prov_103", weekdays, time.Time{}},
	{"prov_106", weekdays, time.Time{}},
	{"prov_107", weekdays[:4], time.Time{}},
	{"prov_108", weekdays, time.Time{}},
	{"prov_109", weekdays, time.Time{}},
	{"prov_111", weekdays[1:], time.Time{}},
	{"prov_110", weekdays, time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC)},
}

// seedTimecards records shifts over the history window. Payroll was run through the end of
// the previous pay period, so the most recent two weeks are still owed.
func seedTimecards(g *ServiceGraph, token string, rng *rand.Rand, today time.Time, summary *SeedSummary) error {
	start := today.AddDate(0, 0, -7*historyWeeks)
	payrollCutoff := today.AddDate(0, 0, -14)
	paid := false
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		if !paid && !d.Before(payrollCutoff) {
			for _, s := range hourlyStaff {
				if err := g.Timecard.PaySalary(token, s.id); err != nil {
					return fmt.Errorf("failed to pay salary for %s: %w", s.id, err)
				}
			}
			paid = true
		}
		for _, s := range hourlyStaff {
			if closedDays[d.Format("2006-01-02")] || !slices.Contains(s.days, d.Weekday()) || (!s.until.IsZero() && d.After(s.until)) {
				continue
			}
			// Occasional sick or vacation day.
			if chance(rng, 0.04) {
				continue
			}
			minutes := int64(8*60 - 15 + rng.IntN(46))
			shift := time.Date(d.Year(), d.Month(), d.Day(), 15, 0, 0, 0, time.UTC).Format(time.RFC3339)
			if err := g.Timecard.CreateManualTimecard(token, s.id, minutes, shift); err != nil {
				return fmt.Errorf("failed to seed timecard for %s: %w", s.id, err)
			}
			summary.TimecardsCount++
		}
	}
	return nil
}
