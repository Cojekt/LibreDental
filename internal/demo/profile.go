package demo

import (
	"fmt"
	"time"
)

// Size selects how much generated practice history a demo save folder holds. Both sizes
// share the same handcrafted records, providers, operatories, plans and removal paths, so
// every screen and edge case is covered either way; only the generated volume differs.
type Size string

const (
	// SizeSmall is the minimum that still exercises every screen, status and edge case:
	// a handful of each kind of record, for everyday development and quick tests. It is
	// not meant to look like a real practice.
	SizeSmall Size = "small"
	// SizeLarge is a busy multi-chair practice, for finding slow queries and UI that
	// doesn't hold up at realistic volume.
	SizeLarge Size = "large"
)

// Sizes lists every supported size, smallest first.
var Sizes = []Size{SizeSmall, SizeLarge}

// ParseSize validates a size name.
func ParseSize(s string) (Size, error) {
	for _, size := range Sizes {
		if string(size) == s {
			return size, nil
		}
	}
	return "", fmt.Errorf("unknown demo data size %q (want one of %v)", s, Sizes)
}

type profile struct {
	patients     int // generated roster, on top of the handcrafted patients
	historyWeeks int // completed visits leading up to the reference day
	futureWeeks  int // bookings in the "upcoming" window a year out (see seedAppointments)
	// fillScale scales how full each chair's day is booked.
	fillScale float64
	// weeklyVisitPerChair books every chair at least once a week however low the fill, so
	// each provider and visit type has history.
	weeklyVisitPerChair bool
	// paperworkPatients is how many generated patients get everyday paperwork documents.
	paperworkPatients int
	// archivedPatients have left the practice (moved, transferred care, deceased).
	archivedPatients int
	// timecardDays limits hourly staff shifts to these weekdays; nil records every shift.
	timecardDays []time.Weekday
	removals     removalCounts
}

// removalCounts is how many records of each kind are created in error and then removed.
type removalCounts struct {
	appointments int // alternates upcoming and past
	conditions   int // first half existing, second half treatment-planned
	claims       int
	payments     int
	documents    int
	optOuts      int // patients who withdrew reminder consent
}

// Both history windows reach back past Memorial Day, the departed hygienist's last week and
// the last payroll run, so those cases stay covered at either size.
var profiles = map[Size]profile{
	SizeSmall: {
		patients: 12, historyWeeks: 4, futureWeeks: 1, fillScale: 0, weeklyVisitPerChair: true,
		paperworkPatients: 2, archivedPatients: 1, timecardDays: []time.Weekday{time.Tuesday},
		removals: removalCounts{appointments: 2, conditions: 2, claims: 1, payments: 1, documents: 1, optOuts: 1},
	},
	SizeLarge: {
		patients: 500, historyWeeks: 6, futureWeeks: 2, fillScale: 1, paperworkPatients: 60, archivedPatients: 15,
		removals: removalCounts{appointments: 10, conditions: 6, claims: 4, payments: 5, documents: 3, optOuts: 5},
	},
}
