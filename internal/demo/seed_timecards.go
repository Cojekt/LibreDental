package demo

import (
	"fmt"
	"math/rand/v2"
	"time"
)

// hourlyStaff clock time through the timecard screen; dentists are paid on production.
var hourlyStaff = []string{"prov_103", "prov_106", "prov_107", "prov_108", "prov_109", "prov_111", "prov_110"}

// seedTimecards records shifts over the history window. Payroll was run through the end of
// the previous pay period, so the most recent two weeks are still owed.
func seedTimecards(g *ServiceGraph, token string, rng *rand.Rand, today time.Time, summary *SeedSummary) error {
	start := today.AddDate(0, 0, -7*historyWeeks)
	payrollCutoff := today.AddDate(0, 0, -14)
	paid := false
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		if !paid && !d.Before(payrollCutoff) {
			for _, s := range hourlyStaff {
				if err := g.Timecard.PaySalary(token, s); err != nil {
					return fmt.Errorf("failed to pay salary for %s: %w", s, err)
				}
			}
			paid = true
		}
		for _, s := range hourlyStaff {
			if closedDays[d.Format("2006-01-02")] || !worksOn(s, d) {
				continue
			}
			// Occasional sick or vacation day.
			if chance(rng, 0.04) {
				continue
			}
			minutes := int64(8*60 - 15 + rng.IntN(46))
			shift := time.Date(d.Year(), d.Month(), d.Day(), 15, 0, 0, 0, time.UTC).Format(time.RFC3339)
			if err := g.Timecard.CreateManualTimecard(token, s, minutes, shift); err != nil {
				return fmt.Errorf("failed to seed timecard for %s: %w", s, err)
			}
			summary.TimecardsCount++
		}
	}
	return nil
}
