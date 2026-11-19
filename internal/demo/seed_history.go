package demo

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// catalogDesc mirrors the US procedure catalog names the generated visits bill against.
var catalogDesc = map[string]string{
	"D0120": "Periodic Oral Evaluation",
	"D0140": "Limited Oral Evaluation - Problem Focused",
	"D0210": "Intraoral - Comprehensive Series of Radiographic Images",
	"D0220": "Intraoral - Periapical First Radiographic Image",
	"D0274": "Bitewings - Four Radiographic Images",
	"D1110": "Prophylaxis - Adult",
	"D1120": "Prophylaxis - Child",
	"D1206": "Topical Application of Fluoride Varnish",
	"D1351": "Dental Sealant - Per Tooth",
	"D2140": "Amalgam - One Surface, Primary or Permanent",
	"D2150": "Amalgam - Two Surfaces, Primary or Permanent",
	"D2391": "Resin-Based Composite - One Surface, Posterior",
	"D2392": "Resin-Based Composite - Two Surfaces, Posterior",
	"D2393": "Resin-Based Composite - Three Surfaces, Posterior",
	"D2740": "Crown - Porcelain/Ceramic Substrate",
	"D2750": "Crown - Porcelain Fused to High Noble Metal",
	"D3310": "Endodontic Therapy - Anterior Tooth",
	"D3320": "Endodontic Therapy - Bicuspid Tooth",
	"D3330": "Endodontic Therapy - Molar Tooth",
	"D4210": "Gingivectomy or Gingivoplasty - Four or More Teeth",
	"D4341": "Periodontal Scaling and Root Planing - Four or More Teeth",
	"D5110": "Complete Denture - Maxillary",
	"D5120": "Complete Denture - Mandibular",
	"D7140": "Extraction, Erupted Tooth or Exposed Root",
	"D7210": "Extraction, Erupted Tooth Requiring Surgical Removal",
}

// coverage is the share of the fee a typical PPO plan pays, by CDT category.
func coverage(code string) int64 {
	switch {
	case code < "D2000":
		return 100
	case code >= "D2700" && code < "D2800", code >= "D5000" && code < "D6000":
		return 50
	default:
		return 80
	}
}

var (
	molars    = []int{2, 3, 14, 15, 18, 19, 30, 31}
	premolars = []int{4, 5, 12, 13, 20, 21, 28, 29}
	anteriors = []int{6, 7, 8, 9, 10, 11, 22, 23, 24, 25, 26, 27}
	posterior = append(slices.Clone(molars), premolars...)
)

type proc struct {
	code     string
	tooth    int
	surfaces []domain.ToothSurface
}

type visit struct {
	reason  string
	minutes int
	procs   []proc
	// planned is a follow-up treatment diagnosed at this visit but not yet scheduled.
	planned []proc
}

type columnKind int

const (
	colHygiene columnKind = iota
	colGeneral
	colEndo
	colPediatric
	colProstho
	colEmergency
)

// column is one provider working one operatory on the days in their staffSchedule.
type column struct {
	provider, operatory, color string
	kind                       columnKind
	fill                       float64
}

var weekdays = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

// staffSchedules is shared by bookings and timecards so the two can't disagree about who
// was in the office on a given day.
var staffSchedules = map[string]struct {
	days  []time.Weekday
	until time.Time // zero = still working here
}{
	"prov_101": {weekdays, time.Time{}},
	"prov_102": {weekdays[:4], time.Time{}},
	"prov_103": {weekdays, time.Time{}},
	"prov_104": {[]time.Weekday{time.Monday, time.Wednesday, time.Thursday, time.Friday}, time.Time{}},
	"prov_105": {weekdays[1:], time.Time{}},
	"prov_106": {weekdays, time.Time{}},
	"prov_107": {weekdays[:4], time.Time{}},
	"prov_108": {weekdays, time.Time{}},
	"prov_109": {weekdays, time.Time{}},
	"prov_110": {weekdays, time.Date(2026, 5, 22, 23, 0, 0, 0, time.UTC)},
	"prov_111": {weekdays[1:], time.Time{}},
}

func worksOn(provider string, d time.Time) bool {
	sch := staffSchedules[provider]
	return slices.Contains(sch.days, d.Weekday()) && (sch.until.IsZero() || !d.After(sch.until))
}

// ageOn is the patient's age in whole years on the given day.
func ageOn(dob, day time.Time) int {
	age := day.Year() - dob.Year()
	if dob.AddDate(age, 0, 0).After(day) {
		age--
	}
	return age
}

// closedDays are weekday holidays inside the generated windows (Memorial Day).
var closedDays = map[string]bool{"2026-05-25": true}

func scheduleColumns() []column {
	return []column{
		{"prov_101", "op_1", "#3b82f6", colGeneral, 0.6},
		{"prov_101", "op_5", "#3b82f6", colEmergency, 0.2},
		{"prov_102", "op_3", "#10b981", colEndo, 0.55},
		{"prov_103", "op_2", "#f59e0b", colHygiene, 0.65},
		{"prov_104", "op_8", "#ec4899", colPediatric, 0.5},
		{"prov_105", "op_4", "#8b5cf6", colProstho, 0.5},
		{"prov_106", "op_6", "#14b8a6", colHygiene, 0.65},
		{"prov_107", "op_7", "#f97316", colHygiene, 0.6},
		{"prov_110", "op_9", "#a3a3a3", colHygiene, 0.6},
	}
}

// historyState carries the generator's running state across days.
type historyState struct {
	g        *ServiceGraph
	profile  profile
	token    string
	rng      *rand.Rand
	plans    demoPlans
	planFees map[string]map[string]int64
	today    time.Time
	loc      *time.Location
	summary  *SeedSummary

	byID      map[string]*domain.Patient
	adults    queue
	minors    queue
	seniors   queue
	blocked   map[string]map[int]bool // patient -> teeth missing or already crowned
	busyDay   map[string]bool         // patient|date already booked
	chairWeek map[string]bool         // provider|operatory|ISO week with at least one booking
	future    map[string]bool         // patients with an upcoming booking
	nextID    int
	payments  []domain.Payment
	claims    []*domain.Claim
}

type queue struct {
	ids []string
	pos int
}

func (q *queue) next(s *historyState, day string) string {
	for range len(q.ids) {
		id := q.ids[q.pos%len(q.ids)]
		q.pos++
		if !s.busyDay[id+"|"+day] {
			return id
		}
	}
	return ""
}

func (s *historyState) id(prefix string) string {
	s.nextID++
	return fmt.Sprintf("%s_%05d", prefix, s.nextID)
}

// seedPracticeHistory simulates the practice's recent weeks: a full schedule across every
// chair, the charting and claims behind each completed visit, and the payments that
// followed, plus an upcoming schedule with its treatment-planned work.
func seedPracticeHistory(g *ServiceGraph, p profile, token string, rng *rand.Rand, plans demoPlans, patients []*domain.Patient, today time.Time, summary *SeedSummary) (*historyState, error) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PDT", -7*3600)
	}
	s := &historyState{
		g: g, profile: p, token: token, rng: rng, plans: plans, planFees: planFeeTable(), today: today, loc: loc, summary: summary,
		byID: map[string]*domain.Patient{}, blocked: map[string]map[int]bool{},
		busyDay: map[string]bool{}, chairWeek: map[string]bool{}, future: map[string]bool{}, nextID: 10000,
	}
	for _, p := range patients {
		s.byID[p.ID] = p
		s.blocked[p.ID] = map[int]bool{}
		switch age := ageOn(p.DateOfBirth, today); {
		case age < 18:
			s.minors.ids = append(s.minors.ids, p.ID)
		default:
			s.adults.ids = append(s.adults.ids, p.ID)
			if age >= 60 {
				s.seniors.ids = append(s.seniors.ids, p.ID)
			}
		}
	}
	for _, q := range []*queue{&s.adults, &s.minors, &s.seniors} {
		rng.Shuffle(len(q.ids), func(i, j int) { q.ids[i], q.ids[j] = q.ids[j], q.ids[i] })
	}

	if err := s.seedExistingCharts(patients); err != nil {
		return nil, err
	}

	start := today.AddDate(0, 0, -7*p.historyWeeks)
	for d := start; d.Before(today); d = d.AddDate(0, 0, 1) {
		if err := s.seedDay(d, false, nil); err != nil {
			return nil, err
		}
	}

	// The handcrafted upcoming bookings (seedAppointments) occupy these days for the
	// original three providers; leave their columns alone so those layouts stay as designed.
	upcoming := today.AddDate(1, 0, 0)
	reserved := map[string]bool{}
	for i := range 3 {
		reserved[upcoming.AddDate(0, 0, i).Format("2006-01-02")] = true
	}
	futureStart := upcoming.AddDate(0, 0, -int(upcoming.Weekday()-time.Monday))
	for d := futureStart; d.Before(futureStart.AddDate(0, 0, 7*p.futureWeeks)); d = d.AddDate(0, 0, 1) {
		var skip []string
		if reserved[d.Format("2006-01-02")] {
			skip = []string{"prov_101", "prov_102", "prov_103"}
		}
		if err := s.seedDay(d, true, skip); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// planFeeTable lists the fees each active plan was seeded with (see seedInsurancePlans).
func planFeeTable() map[string]map[string]int64 {
	out := map[string]map[string]int64{}
	for _, p := range insurancePlans {
		if !p.archived {
			out[p.name] = p.fees
		}
	}
	return out
}

func (s *historyState) seedDay(d time.Time, upcoming bool, skip []string) error {
	day := d.Format("2006-01-02")
	if closedDays[day] {
		return nil
	}
	for _, col := range scheduleColumns() {
		if !worksOn(col.provider, d) || slices.Contains(skip, col.provider) {
			continue
		}
		fill := col.fill * s.profile.fillScale
		if upcoming {
			// Near-term days are nearly full; later weeks still have openings.
			week := int(d.Sub(s.today.AddDate(1, 0, 0)).Hours() / 24 / 7)
			fill *= []float64{1.0, 0.6}[max(0, min(week, 1))]
		}
		year, week := d.ISOWeek()
		chairWeek := fmt.Sprintf("%s|%s|%d-%d", col.provider, col.operatory, year, week)
		t := 8 * 60
		for t < 17*60 {
			if t >= 12*60 && t < 13*60 {
				t = 13 * 60
			}
			force := s.profile.weeklyVisitPerChair && !s.chairWeek[chairWeek]
			if !force && !chance(s.rng, fill) {
				t += 30
				continue
			}
			pid := s.pickPatient(col.kind, day)
			if pid == "" {
				break
			}
			v := s.planVisit(col.kind, pid)
			end := t + v.minutes
			if t < 12*60 && end > 12*60 {
				t = 13 * 60
				end = t + v.minutes
			}
			if end > 17*60 {
				break
			}
			if err := s.book(col, d, t, end, pid, v, upcoming); err != nil {
				return err
			}
			s.busyDay[pid+"|"+day] = true
			s.chairWeek[chairWeek] = true
			t = end
		}
	}
	return nil
}

func (s *historyState) pickPatient(kind columnKind, day string) string {
	switch kind {
	case colPediatric:
		return s.minors.next(s, day)
	case colHygiene:
		if chance(s.rng, 0.2) {
			return s.minors.next(s, day)
		}
	case colProstho:
		if chance(s.rng, 0.3) {
			return s.seniors.next(s, day)
		}
	}
	return s.adults.next(s, day)
}

func (s *historyState) isMinor(pid string) bool {
	return ageOn(s.byID[pid].DateOfBirth, s.today) < 18
}

// tooth picks an unblocked tooth from the candidates, or 0 if none is left.
func (s *historyState) tooth(pid string, from []int) int {
	for _, i := range s.rng.Perm(len(from)) {
		if !s.blocked[pid][from[i]] {
			return from[i]
		}
	}
	return 0
}

func (s *historyState) filling(pid string) proc {
	t := s.tooth(pid, posterior)
	if t == 0 {
		return proc{code: "D0140"}
	}
	O, M, D := domain.SurfaceOcclusal, domain.SurfaceMesial, domain.SurfaceDistal
	switch n := s.rng.IntN(10); {
	case n < 1:
		return proc{"D2150", t, []domain.ToothSurface{M, O}}
	case n < 4:
		return proc{"D2391", t, []domain.ToothSurface{O}}
	case n < 8:
		return proc{"D2392", t, []domain.ToothSurface{pick(s.rng, []domain.ToothSurface{M, D}), O}}
	default:
		return proc{"D2393", t, []domain.ToothSurface{M, O, D}}
	}
}

func endoCode(tooth int) string {
	switch {
	case slices.Contains(molars, tooth):
		return "D3330"
	case slices.Contains(premolars, tooth):
		return "D3320"
	default:
		return "D3310"
	}
}

func (s *historyState) planVisit(kind columnKind, pid string) visit {
	rng := s.rng
	minor := s.isMinor(pid)
	switch kind {
	case colHygiene:
		if minor {
			return visit{"Child Prophylaxis & Fluoride", 45, []proc{{code: "D0120"}, {code: "D1120"}, {code: "D1206"}}, nil}
		}
		switch n := rng.IntN(100); {
		case n < 8:
			return visit{pick(rng, []string{"Scaling & Root Planing - Upper Right/Lower Right", "Scaling & Root Planing - Upper Left/Lower Left"}), 90, []proc{{code: "D4341"}}, nil}
		case n < 14:
			return visit{"New Patient Exam, Full Series X-Rays & Cleaning", 90, []proc{{code: "D0120"}, {code: "D0210"}, {code: "D1110"}}, s.diagnose(pid, 0.6)}
		case n < 50:
			return visit{"Recall Exam, Bitewings & Prophylaxis", 60, []proc{{code: "D0120"}, {code: "D0274"}, {code: "D1110"}}, s.diagnose(pid, 0.25)}
		default:
			return visit{"Periodic Exam & Prophylaxis", 60, []proc{{code: "D0120"}, {code: "D1110"}}, s.diagnose(pid, 0.15)}
		}
	case colEmergency:
		v := visit{pick(rng, []string{"Emergency: Toothache", "Emergency: Broken Tooth", "Emergency: Swelling", "Emergency: Lost Filling"}), 30, []proc{{code: "D0140"}, {code: "D0220"}}, nil}
		if t := s.tooth(pid, posterior); t != 0 && chance(rng, 0.6) {
			v.planned = []proc{{code: endoCode(t), tooth: t, surfaces: []domain.ToothSurface{domain.SurfaceOcclusal}}}
		}
		return v
	case colEndo:
		if t := s.tooth(pid, append(slices.Clone(posterior), anteriors...)); t != 0 && chance(s.rng, 0.6) {
			return visit{fmt.Sprintf("Root Canal Therapy #%d", t), 90, []proc{{code: endoCode(t), tooth: t, surfaces: []domain.ToothSurface{domain.SurfaceOcclusal}}},
				[]proc{{code: "D2740", tooth: t}}}
		}
		if t := s.tooth(pid, []int{1, 16, 17, 32}); t != 0 {
			return visit{fmt.Sprintf("Surgical Extraction #%d (Third Molar)", t), 60, []proc{{code: "D7210", tooth: t}}, nil}
		}
		if t := s.tooth(pid, posterior); t != 0 {
			return visit{fmt.Sprintf("Extraction #%d", t), 45, []proc{{code: "D7140", tooth: t}}, nil}
		}
	case colPediatric:
		if chance(rng, 0.45) {
			var ps []proc
			for _, t := range []int{3, 14, 19, 30} {
				if !s.blocked[pid][t] && chance(rng, 0.7) {
					ps = append(ps, proc{"D1351", t, []domain.ToothSurface{domain.SurfaceOcclusal}})
				}
			}
			if len(ps) > 0 {
				return visit{"Sealants - Permanent Molars", 45, ps, nil}
			}
		}
		if chance(rng, 0.15) {
			return visit{"Limited Exam - Tooth Pain", 30, []proc{{code: "D0140"}, {code: "D0220"}}, nil}
		}
		f := s.filling(pid)
		if f.code == "D2150" {
			f = proc{"D2391", f.tooth, []domain.ToothSurface{domain.SurfaceOcclusal}}
		}
		return visit{"Pediatric Restorative", 45, []proc{f}, nil}
	case colProstho:
		age := ageOn(s.byID[pid].DateOfBirth, s.today)
		if age >= 60 && chance(rng, 0.25) {
			code := pick(rng, []string{"D5110", "D5120"})
			return visit{"Complete Denture - Final Impressions & Delivery", 60, []proc{{code: code}}, nil}
		}
		if t := s.tooth(pid, posterior); t != 0 && chance(rng, 0.8) {
			return visit{fmt.Sprintf("Crown Preparation #%d", t), 90, []proc{{code: pick(rng, []string{"D2740", "D2740", "D2750"}), tooth: t}}, nil}
		}
		return visit{"Prosthodontic Consultation", 30, []proc{{code: "D0140"}}, nil}
	case colGeneral:
		switch n := rng.IntN(100); {
		case n < 55:
			ps := []proc{s.filling(pid)}
			if chance(rng, 0.35) {
				if f := s.filling(pid); f.tooth != ps[0].tooth && f.tooth != 0 {
					ps = append(ps, f)
				}
			}
			return visit{"Composite Restorations", 60, ps, nil}
		case n < 75:
			if t := s.tooth(pid, molars); t != 0 {
				return visit{fmt.Sprintf("Crown Preparation #%d", t), 90, []proc{{code: "D2740", tooth: t}}, nil}
			}
		case n < 85:
			if t := s.tooth(pid, posterior); t != 0 {
				return visit{fmt.Sprintf("Extraction #%d", t), 45, []proc{{code: "D7140", tooth: t}}, nil}
			}
		case n < 92:
			return visit{"Gingivectomy - Lower Anterior", 60, []proc{{code: "D4210"}}, nil}
		}
		return visit{"Limited Exam & Periapical X-Ray", 30, []proc{{code: "D0140"}, {code: "D0220"}}, s.diagnose(pid, 0.5)}
	}
	return visit{"Periodic Exam", 30, []proc{{code: "D0120"}}, nil}
}

// diagnose returns, with probability p, a filling the exam found that still needs booking.
func (s *historyState) diagnose(pid string, p float64) []proc {
	if s.isMinor(pid) || !chance(s.rng, p) {
		return nil
	}
	if f := s.filling(pid); f.tooth != 0 {
		return []proc{f}
	}
	return nil
}

var (
	completedNotes = []string{
		"", "", "Patient tolerated procedure well.", "No complications. Post-op instructions given.",
		"Local anesthetic: 2% lidocaine 1:100k epi, 1 carpule.", "Reviewed home care; recommended electric toothbrush.",
		"Moderate calculus lower anteriors.", "Patient reports sensitivity to cold; monitor.",
	}
	cancelNotes = []string{
		"Patient called to cancel; will reschedule.", "Cancelled - patient sick.", "Cancelled - work conflict.",
		"Office cancelled - provider out. Patient rebooked.", "Cancelled via voicemail the night before.",
	}
	upcomingNotes = []string{"", "", "", "Confirmed via SMS.", "Left voicemail to confirm.", "Patient requested reminder call.", "Pre-medicate per medical history."}
)

func (s *historyState) book(col column, d time.Time, startMin, endMin int, pid string, v visit, upcoming bool) error {
	at := func(m int) time.Time {
		return time.Date(d.Year(), d.Month(), d.Day(), m/60, m%60, 0, 0, s.loc)
	}
	start := at(startMin)
	created := start.AddDate(0, 0, -(3 + s.rng.IntN(60)))
	if created.After(s.today) {
		created = s.today.AddDate(0, 0, -s.rng.IntN(30))
	}
	a := &domain.Appointment{
		ID:          s.id("appt"),
		PatientID:   pid,
		ProviderID:  col.provider,
		OperatoryID: col.operatory,
		StartTime:   start,
		EndTime:     at(endMin),
		Reason:      v.reason,
		Color:       col.color,
		CreatedAt:   created,
		UpdatedAt:   created,
		Version:     1,
	}
	switch n := s.rng.IntN(100); {
	case upcoming:
		a.Status = domain.AppointmentStatusScheduled
		if d.Before(s.today.AddDate(1, 0, 7)) && n < 60 {
			a.Status = domain.AppointmentStatusConfirmed
		}
		a.Notes = pick(s.rng, upcomingNotes)
	case n < 5:
		a.Status, a.Notes = domain.AppointmentStatusCancelled, pick(s.rng, cancelNotes)
	case n < 9:
		a.Status, a.Notes = domain.AppointmentStatusNoShow, "No-show; no call. Front desk to follow up."
	default:
		a.Status, a.Notes = domain.AppointmentStatusComplete, pick(s.rng, completedNotes)
	}
	if _, err := s.g.Appointment.CreateAppointment(s.token, a); err != nil {
		return fmt.Errorf("failed to seed appointment %s: %w", a.ID, err)
	}
	s.summary.AppointmentsCount++

	switch a.Status {
	case domain.AppointmentStatusComplete:
		return s.completeVisit(a, d, v)
	case domain.AppointmentStatusScheduled, domain.AppointmentStatusConfirmed:
		s.future[pid] = true
		// Upcoming restorative work is already on the chart as treatment-planned.
		for _, p := range v.procs {
			if p.tooth != 0 {
				if _, err := s.chart(pid, col.provider, p, domain.ToothStatusTreatmentPlanned); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// price confirms a fee the way the pricing review does: the patient's plan fee when the
// plan lists the code, otherwise the clinician's standard fee.
func (s *historyState) price(pid, clinician, code string) (*domain.PricingSource, error) {
	carrier := s.byID[pid].InsuranceCarrier
	if _, ok := s.planFees[carrier][code]; ok {
		return demoPrice{kind: "insurance", clinician: clinician, plan: carrier}.preview(s.g, s.token, s.plans, pid, code)
	}
	return demoPrice{kind: "standard", clinician: clinician}.preview(s.g, s.token, s.plans, pid, code)
}

func (s *historyState) chart(pid, clinician string, p proc, status domain.ToothStatus) (*domain.SaveToothConditionPayload, error) {
	c := &domain.SaveToothConditionPayload{
		ID: s.id("cond"), PatientID: pid, ToothNumber: p.tooth, Surfaces: p.surfaces,
		ADACode: p.code, Description: catalogDesc[p.code], Status: status,
	}
	if c.Surfaces == nil {
		c.Surfaces = []domain.ToothSurface{}
	}
	src, err := s.price(pid, clinician, p.code)
	if err != nil {
		return nil, err
	}
	c.Fee, c.Pricing = src.Amount, src
	if _, err := s.g.Chart.SaveToothCondition(s.token, c); err != nil {
		return nil, fmt.Errorf("failed to seed tooth condition %s: %w", c.ID, err)
	}
	s.summary.ConditionsCount++
	switch {
	case p.code == "D7140" || p.code == "D7210":
		s.blocked[pid][p.tooth] = true
	case p.code >= "D2700" && p.code < "D2800":
		s.blocked[pid][p.tooth] = true
	}
	return c, nil
}

var (
	rejectReasons = []string{
		"Denied: frequency limitation - prophylaxis covered twice per benefit year.",
		"Denied: missing tooth clause applies.",
		"Denied: additional documentation (narrative and x-ray) requested.",
		"Denied: patient not eligible on date of service. Verify coverage.",
	}
	patientMethods = []domain.PaymentMethod{domain.PaymentMethodCreditCard, domain.PaymentMethodCreditCard, domain.PaymentMethodCreditCard, domain.PaymentMethodCash, domain.PaymentMethodCheck}
)

func dateAfter(d time.Time, days int, limit time.Time) string {
	out := d.AddDate(0, 0, days)
	if out.After(limit) {
		out = limit
	}
	return out.Format("2006-01-02")
}

func (s *historyState) completeVisit(a *domain.Appointment, d time.Time, v visit) error {
	pid := a.PatientID
	patient := s.byID[pid]
	insured := patient.InsuranceCarrier != ""
	dos := d.Format("2006-01-02")
	claim := &domain.Claim{
		ID: s.id("claim"), PatientID: pid, ProviderID: a.ProviderID, AppointmentID: a.ID,
		DateOfService: dos, CreatedAt: d, UpdatedAt: d,
	}
	var allowed, portion int64
	for _, p := range v.procs {
		li := domain.ClaimLineItem{ID: s.id("li"), ADACode: p.code, Description: catalogDesc[p.code], Surfaces: []domain.ToothSurface{}}
		if p.tooth != 0 {
			c, err := s.chart(pid, a.ProviderID, p, domain.ToothStatusCompleted)
			if err != nil {
				return err
			}
			li.ToothConditionID, li.ToothNumber, li.Surfaces, li.Fee = c.ID, c.ToothNumber, c.Surfaces, c.Fee
		} else {
			src, err := s.price(pid, a.ProviderID, p.code)
			if err != nil {
				return err
			}
			li.Fee, li.Pricing = src.Amount, src
		}
		li.PatientPortion = li.Fee
		if insured {
			li.InsuranceAllowed = li.Fee * coverage(p.code) / 100
			li.PatientPortion = li.Fee - li.InsuranceAllowed
		}
		allowed += li.InsuranceAllowed
		portion += li.PatientPortion
		claim.LineItems = append(claim.LineItems, li)
	}
	for _, p := range v.planned {
		if !s.blocked[pid][p.tooth] {
			if _, err := s.chart(pid, a.ProviderID, p, domain.ToothStatusTreatmentPlanned); err != nil {
				return err
			}
		}
	}

	age := int(s.today.Sub(d).Hours() / 24)
	paidAtVisit := chance(s.rng, 0.8)
	var pays []domain.Payment
	addPay := func(amount int64, m domain.PaymentMethod, date, note string) {
		if amount > 0 {
			pays = append(pays, domain.Payment{Amount: amount, Method: m, Date: date, Notes: note})
		}
	}
	if !insured {
		claim.Notes = "Self-pay patient; statement issued at checkout."
		claim.Status = domain.ClaimStatusAccepted
		switch {
		case paidAtVisit:
			claim.Status = domain.ClaimStatusPaid
			addPay(portion, pick(s.rng, patientMethods), dos, "Paid in full at checkout")
		case chance(s.rng, 0.5):
			addPay(portion/2, domain.PaymentMethodCreditCard, dos, "Payment plan - first installment")
		}
	} else {
		claim.Status = s.claimStatus(age)
		if paidAtVisit {
			addPay(portion, pick(s.rng, patientMethods), dos, "Estimated patient portion collected at checkout")
		}
		switch claim.Status {
		case domain.ClaimStatusPaid:
			eob := fmt.Sprintf("EOB #%06d", s.rng.IntN(1000000))
			claim.Notes = "Processed and paid. " + eob
			addPay(allowed, domain.PaymentMethodInsurance, dateAfter(d, 14+s.rng.IntN(21), s.today), patient.InsuranceCarrier+" payment "+eob)
			if !paidAtVisit && age > 30 && chance(s.rng, 0.4) {
				addPay(portion, domain.PaymentMethodWriteOff, dateAfter(d, 30, s.today), "Courtesy adjustment approved by office manager")
			}
		case domain.ClaimStatusRejected:
			claim.Notes = pick(s.rng, rejectReasons)
		case domain.ClaimStatusAccepted:
			claim.Notes = "Accepted by payer; awaiting remittance."
		case domain.ClaimStatusSubmitted:
			claim.Notes = "Submitted electronically."
		default:
			claim.Notes = "Ready for review before submission."
		}
	}

	if _, err := s.g.Billing.CreateClaim(s.token, claim); err != nil {
		return fmt.Errorf("failed to seed claim %s: %w", claim.ID, err)
	}
	s.summary.ClaimsCount++
	s.claims = append(s.claims, claim)

	for _, p := range pays {
		p.ID, p.PatientID, p.ClaimID = s.id("pay"), pid, claim.ID
		date, _ := time.Parse("2006-01-02", p.Date)
		p.CreatedAt = date.Add(17 * time.Hour)
		if _, err := s.g.Billing.RecordPayment(s.token, &p); err != nil {
			return fmt.Errorf("failed to seed payment %s: %w", p.ID, err)
		}
		s.summary.PaymentsCount++
		s.payments = append(s.payments, p)
	}
	return nil
}

// claimStatus follows a claim's typical progress by how many days ago the visit was.
func (s *historyState) claimStatus(age int) domain.ClaimStatus {
	n := s.rng.IntN(100)
	switch {
	case age > 30:
		switch {
		case n < 84:
			return domain.ClaimStatusPaid
		case n < 92:
			return domain.ClaimStatusRejected
		default:
			return domain.ClaimStatusAccepted
		}
	case age > 10:
		switch {
		case n < 25:
			return domain.ClaimStatusPaid
		case n < 55:
			return domain.ClaimStatusAccepted
		case n < 60:
			return domain.ClaimStatusRejected
		default:
			return domain.ClaimStatusSubmitted
		}
	default:
		if n < 55 {
			return domain.ClaimStatusDraft
		}
		return domain.ClaimStatusSubmitted
	}
}

// seedExistingCharts records the work patients arrived with: prior restorations, crowns and
// missing teeth. These are history, not billable treatment, so they carry no pricing.
func (s *historyState) seedExistingCharts(patients []*domain.Patient) error {
	for _, p := range patients {
		age := s.today.Year() - p.DateOfBirth.Year()
		if age < 18 {
			continue
		}
		var items []proc
		var missing []int
		for _, t := range []int{1, 16, 17, 32} {
			if chance(s.rng, 0.45) {
				missing = append(missing, t)
			}
		}
		if age > 55 && chance(s.rng, 0.4) {
			missing = append(missing, pick(s.rng, []int{3, 14, 19, 30}))
		}
		for _, t := range missing {
			if !s.blocked[p.ID][t] {
				s.blocked[p.ID][t] = true
				items = append(items, proc{tooth: t})
			}
		}
		for range s.rng.IntN(min(6, 1+age/12)) {
			f := s.filling(p.ID)
			if f.tooth == 0 {
				break
			}
			if age > 45 && chance(s.rng, 0.25) {
				f = proc{code: pick(s.rng, []string{"D2740", "D2750"}), tooth: f.tooth}
				s.blocked[p.ID][f.tooth] = true
			}
			items = append(items, f)
		}
		for _, it := range items {
			c := &domain.SaveToothConditionPayload{
				ID: s.id("cond"), PatientID: p.ID, ToothNumber: it.tooth, Surfaces: it.surfaces,
				ADACode: it.code, Description: catalogDesc[it.code], Status: domain.ToothStatusExisting,
			}
			if it.code == "" {
				c.Status, c.Description = domain.ToothStatusMissing, "Missing Tooth"
			}
			if c.Surfaces == nil {
				c.Surfaces = []domain.ToothSurface{}
			}
			if _, err := s.g.Chart.SaveToothCondition(s.token, c); err != nil {
				return fmt.Errorf("failed to seed existing condition %s: %w", c.ID, err)
			}
			s.summary.ConditionsCount++
		}
	}
	return nil
}
