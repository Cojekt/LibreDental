package demo

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

var (
	maleNames = []string{
		"James", "Michael", "Daniel", "Luis", "Wei", "Arjun", "Omar", "Ethan", "Noah", "Mateo",
		"Hiroshi", "Andre", "Samuel", "Kwame", "Diego", "Brian", "Patrick", "Minh", "Elijah", "Tariq",
		"Gregory", "Isaac", "Rafael", "Victor", "Aaron", "Jamal", "Connor", "Nikhil", "Felipe", "Leo",
	}
	femaleNames = []string{
		"Olivia", "Ava", "Isabella", "Grace", "Mei", "Priya", "Fatima", "Chloe", "Lucia", "Hannah",
		"Yuki", "Amara", "Rosa", "Leah", "Nadia", "Emily", "Sienna", "Thi", "Zoe", "Camila",
		"Margaret", "Ruth", "Aaliyah", "Ingrid", "Valentina", "Sara", "Keiko", "Abigail", "Maya", "Elena",
	}
	lastNames = []string{
		"Nguyen", "Garcia", "Smith", "Johnson", "Patel", "Kim", "Lopez", "Brown", "Wong", "Hernandez",
		"Davis", "Wilson", "Anderson", "Thompson", "Rodriguez", "Lee", "Walker", "Singh", "Okonkwo", "Murphy",
		"Rossi", "Kowalski", "Haddad", "Ivanova", "Yamamoto", "Clark", "Lewis", "Reyes", "Scott", "Bennett",
		"Huang", "Flores", "Morales", "Fischer", "Larsen", "Mensah", "Sullivan", "Gupta", "Torres", "Castillo",
		"Park", "Ramirez", "Bailey", "Coleman", "Ahmed", "Jensen", "Novak", "Ortiz", "Price", "Delgado",
	}
	streets = []string{
		"Valencia Street", "Geary Boulevard", "Irving Street", "Divisadero Street", "Clement Street",
		"Taraval Street", "Noriega Street", "Folsom Street", "Hayes Street", "Fillmore Street",
		"Ocean Avenue", "Judah Street", "Castro Street", "Polk Street", "Grand Avenue",
		"Telegraph Avenue", "El Camino Real", "Mission Street", "Lombard Street", "Cortland Avenue",
	}
	cities = []struct{ name, zip string }{
		{"San Francisco", "94110"}, {"San Francisco", "94122"}, {"San Francisco", "94115"},
		{"San Francisco", "94117"}, {"San Francisco", "94112"}, {"San Francisco", "94116"},
		{"Daly City", "94014"}, {"South San Francisco", "94080"}, {"Oakland", "94610"},
		{"Berkeley", "94703"}, {"San Mateo", "94401"}, {"Burlingame", "94010"},
	}
	referralSources = []string{
		"Online Search", "Word of Mouth", "Doctor Referral", "Insurance Directory",
		"Social Media", "Walk-in", "Employer Benefits Fair", "Existing Patient Referral",
	}
	medicalAlertPool = []string{
		"High Blood Pressure", "Type II Diabetes", "Asthma", "Heart Murmur", "Pregnancy",
		"Artificial Joint (Hip) - Premedicate", "Anticoagulant Therapy (Warfarin)", "Bisphosphonate Use",
		"History of Rheumatic Fever", "Anxiety - Prefers Nitrous", "Sleep Apnea", "Hepatitis C (Resolved)",
	}
	allergyPool = []string{"Penicillin", "Latex", "Codeine", "Sulfa Drugs", "Ibuprofen", "Erythromycin", "Chlorhexidine"}
	adultNotes  = []string{
		"", "", "", "Prefers morning appointments.", "Prefers late-afternoon appointments.",
		"Dental anxiety; explain each step.", "Gag reflex - use smaller sensor for x-rays.",
		"Works nights; call after 11am.", "Requests text reminders only.", "Interested in Invisalign consult.",
		"Bruxism - recommended night guard.", "Needs wheelchair-accessible operatory.",
	}
)

var nicknames = map[string]string{
	"James": "Jim", "Michael": "Mike", "Daniel": "Dan", "Samuel": "Sam", "Gregory": "Greg",
	"Elijah": "Eli", "Margaret": "Peggy", "Abigail": "Abby", "Valentina": "Vale", "Isabella": "Bella",
}

// carrierMix weights the payer mix of the roster; an empty carrier is a self-pay patient.
var carrierMix = []struct {
	carrier, prefix, group string
	weight                 int
}{
	{"Delta Dental PPO", "DEL", "GRP-10023", 25},
	{"MetLife Dental", "MET", "GRP-88102", 12},
	{"Cigna Dental Health", "CGN", "GRP-44012", 12},
	{"Aetna Dental", "AET", "GRP-55102", 10},
	{"Guardian Dental", "GRD", "GRP-11045", 8},
	{"Humana Dental Choice", "HUM", "GRP-77201", 5},
	{"United Concordia", "UCC", "GRP-31870", 5},
	{"", "", "", 23},
}

func pick[T any](rng *rand.Rand, xs []T) T { return xs[rng.IntN(len(xs))] }

func chance(rng *rand.Rand, p float64) bool { return rng.Float64() < p }

func phone(rng *rand.Rand) string {
	return fmt.Sprintf("(555) %03d-%04d", 200+rng.IntN(800), rng.IntN(10000))
}

// fakeSSN uses the 9xx area range, which is never issued as a real Social Security number.
func fakeSSN(rng *rand.Rand) string {
	return fmt.Sprintf("9%02d-%02d-%04d", rng.IntN(100), 70+rng.IntN(30), rng.IntN(10000))
}

func pickCarrier(rng *rand.Rand) int {
	n := rng.IntN(100)
	for i, c := range carrierMix {
		if n < c.weight {
			return i
		}
		n -= c.weight
	}
	return len(carrierMix) - 1
}

func sample(rng *rand.Rand, pool []string, max int) []string {
	out := []string{}
	for _, i := range rng.Perm(len(pool))[:rng.IntN(max+1)] {
		out = append(out, pool[i])
	}
	return out
}

func birthDate(rng *rand.Rand, today time.Time, minAge, maxAge int) time.Time {
	age := minAge + rng.IntN(maxAge-minAge+1)
	d := today.AddDate(-age, 0, -rng.IntN(365))
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// generatePatients builds households (a primary adult, sometimes a partner and children
// sharing an address, guarantor and policy) until the roster reaches its target size.
func generatePatients(rng *rand.Rand, today time.Time, count int) []*domain.Patient {
	var out []*domain.Patient
	next := 1001
	newPatient := func(first, last string, sex domain.Sex, dob time.Time) *domain.Patient {
		// Patients joined the practice at various points over the last eight years.
		created := today.AddDate(0, 0, -30-rng.IntN(8*365))
		if created.Before(dob) {
			created = dob.AddDate(0, 1, 0)
		}
		p := &domain.Patient{
			ID:                     fmt.Sprintf("pat_%04d", next),
			FirstName:              first,
			LastName:               last,
			DateOfBirth:            dob,
			Sex:                    sex,
			PreferredContactMethod: pick(rng, []string{"phone", "sms", "sms", "email"}),
			PreferredLanguage:      "en",
			ReminderOptIn:          chance(rng, 0.85),
			ReferralSource:         pick(rng, referralSources),
			CountryCode:            domain.CountryUS,
			MedicalAlerts:          []string{},
			Allergies:              []string{},
			Status:                 domain.StatusActive,
			CreatedAt:              created,
			UpdatedAt:              created,
			Version:                1,
		}
		next++
		return p
	}

	for len(out) < count {
		last := pick(rng, lastNames)
		city := pick(rng, cities)
		addr1 := fmt.Sprintf("%d %s", 100+rng.IntN(4800), pick(rng, streets))
		addr2 := ""
		if chance(rng, 0.25) {
			addr2 = fmt.Sprintf("Apt %d%c", 1+rng.IntN(12), 'A'+rune(rng.IntN(4)))
		}
		homePhone := phone(rng)
		lang := pick(rng, []string{"en", "en", "en", "en", "en", "en", "en", "es", "es", "zh", "vi"})
		ci := pickCarrier(rng)
		carrier := carrierMix[ci]
		policy := ""
		if carrier.carrier != "" {
			policy = fmt.Sprintf("%s-%07d", carrier.prefix, rng.IntN(10000000))
		}

		var household []*domain.Patient
		sex := pick(rng, []domain.Sex{domain.SexMale, domain.SexFemale})
		primary := newPatient(nameFor(rng, sex), last, sex, birthDate(rng, today, 22, 88))
		household = append(household, primary)

		var partner *domain.Patient
		if chance(rng, 0.35) {
			psex := domain.SexFemale
			if sex == domain.SexFemale {
				psex = domain.SexMale
			}
			if chance(rng, 0.08) {
				psex = sex
			}
			partner = newPatient(nameFor(rng, psex), last, psex, birthDate(rng, today, 22, 80))
			household = append(household, partner)
		}
		// birthDate can back-date up to a year past the whole-year age, so capping the child at
		// the primary's age minus 19 keeps the primary at least 18 when each child was born.
		maxKidAge := min(17, ageOn(primary.DateOfBirth, today)-19)
		kids := 0
		if maxKidAge >= 4 && primary.DateOfBirth.After(today.AddDate(-55, 0, 0)) && chance(rng, 0.35) {
			kids = 1 + rng.IntN(3)
		}
		for range kids {
			ksex := pick(rng, []domain.Sex{domain.SexMale, domain.SexFemale})
			household = append(household, newPatient(nameFor(rng, ksex), last, ksex, birthDate(rng, today, 4, maxKidAge)))
		}

		primaryName := primary.FirstName + " " + primary.LastName
		for i, p := range household {
			minor := p.DateOfBirth.After(today.AddDate(-18, 0, 0))
			p.AddressLine1, p.AddressLine2 = addr1, addr2
			p.City, p.StateProvince, p.PostalCode = city.name, "CA", city.zip
			p.PreferredLanguage = lang
			p.PreferredProviderID = pick(rng, []string{"prov_101", "prov_101", "prov_105", "prov_103", "prov_106", "prov_107"})
			p.InsuranceCarrier, p.InsurancePolicyNumber = carrier.carrier, policy
			if carrier.carrier != "" {
				p.InsuranceGroupNumber = carrier.group
			}
			p.InsuranceIsSubscriber = i == 0 && carrier.carrier != ""

			if minor {
				p.PhonePrimary = homePhone
				p.Email = ""
				p.PreferredProviderID = "prov_104"
				p.GuarantorName, p.GuarantorRel, p.GuarantorPhone = primaryName, "Parent", homePhone
				p.EmergencyContactName, p.EmergencyContactRel, p.EmergencyContactPhone = primaryName, "Parent", homePhone
				if p.DateOfBirth.Before(today.AddDate(-12, 0, 0)) && chance(rng, 0.4) {
					p.Notes = "Orthodontic evaluation recommended."
				}
			} else {
				p.PhonePrimary = phone(rng)
				if chance(rng, 0.3) {
					p.PhoneSecondary = homePhone
				}
				p.Email = fmt.Sprintf("%s.%s%d@%s", strings.ToLower(p.FirstName), strings.ToLower(strings.ReplaceAll(p.LastName, "'", "")),
					rng.IntN(100), pick(rng, []string{"example.com", "example.org", "example.net"}))
				p.GuarantorName, p.GuarantorRel, p.GuarantorPhone = "Self", "Self", p.PhonePrimary
				if p.DateOfBirth.Before(today.AddDate(-40, 0, 0)) {
					pool := medicalAlertPool
					if p.Sex != domain.SexFemale || p.DateOfBirth.Before(today.AddDate(-50, 0, 0)) {
						pool = slices.DeleteFunc(slices.Clone(pool), func(a string) bool { return a == "Pregnancy" })
					}
					p.MedicalAlerts = sample(rng, pool, 2)
				}
				p.Allergies = sample(rng, allergyPool, 1)
				p.Notes = pick(rng, adultNotes)
				if chance(rng, 0.6) {
					p.NationalIDType, p.NationalID = "ssn", fakeSSN(rng)
				}
				p.PreferredName = nicknames[p.FirstName]
			}

			switch {
			case p == partner:
				p.EmergencyContactName, p.EmergencyContactRel, p.EmergencyContactPhone = primaryName, "Spouse", primary.PhonePrimary
			case p == primary && partner != nil:
				// The phone is filled in after the loop, once the partner's has been generated.
				p.EmergencyContactName, p.EmergencyContactRel = partner.FirstName+" "+partner.LastName, "Spouse"
			case p == primary:
				p.EmergencyContactName = nameFor(rng, pick(rng, []domain.Sex{domain.SexMale, domain.SexFemale})) + " " + pick(rng, lastNames)
				p.EmergencyContactRel = pick(rng, []string{"Sibling", "Friend", "Parent", "Adult Child"})
				p.EmergencyContactPhone = phone(rng)
			}

			// Dependents on the primary's policy record the policyholder.
			if carrier.carrier != "" && i > 0 {
				rel := domain.SubscriberRelationshipChild
				if p == partner {
					rel = domain.SubscriberRelationshipSpouse
				}
				p.InsuranceSubscriberFirstName, p.InsuranceSubscriberLastName = primary.FirstName, primary.LastName
				p.InsuranceSubscriberDOB = primary.DateOfBirth.Format("2006-01-02")
				p.InsuranceSubscriberSex = primary.Sex
				p.InsuranceSubscriberRelationship = rel
				p.InsuranceSubscriberAddressLine1, p.InsuranceSubscriberAddressLine2 = addr1, addr2
				p.InsuranceSubscriberCity, p.InsuranceSubscriberState, p.InsuranceSubscriberPostalCode = city.name, "CA", city.zip
			}
			out = append(out, p)
		}
		if partner != nil {
			primary.EmergencyContactPhone = partner.PhonePrimary
		}
	}
	return out[:count]
}

func nameFor(rng *rand.Rand, sex domain.Sex) string {
	if sex == domain.SexFemale {
		return pick(rng, femaleNames)
	}
	return pick(rng, maleNames)
}
