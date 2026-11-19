package demo

import (
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
)

// bootstrapFirstProvider creates the very first provider through the same unauthenticated
// onboarding path a fresh install uses, which also opens a session as that provider.
// The returned token is used to authenticate every subsequent seed step.
func bootstrapFirstProvider(g *ServiceGraph, now time.Time, summary *SeedSummary) (string, error) {
	first := domain.Provider{
		ID:            "prov_101",
		Name:          "Dr. Sarah Jenkins",
		Role:          domain.RoleDentist,
		Specialty:     "General Dentistry & Restorative",
		LicenseNumber: "DEN-98212",
		Email:         "s.jenkins@apexdentalstudio.com",
		Phone:         "555-0101",
		Color:         "#3b82f6",
		Pin:           "1111",
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	token, err := g.Practice.CreateInitialProvider(first)
	if err != nil {
		return "", fmt.Errorf("failed to bootstrap first provider: %w", err)
	}
	summary.ProvidersCount++
	return token, nil
}

// initPracticeConfig sets the practice country unauthenticated, as onboarding step one does.
func initPracticeConfig(g *ServiceGraph) (*domain.PracticeConfig, error) {
	cfg, err := g.Practice.SetConfig("", string(domain.CountryUS))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize practice config: %w", err)
	}
	return cfg, nil
}

func seedPracticeConfig(g *ServiceGraph, token string, cfg *domain.PracticeConfig, now time.Time, summary *SeedSummary) error {

	cfg.ClinicName = "Apex Dental Studio"
	cfg.Tagline = "Modern Dental Care & Implant Center"
	cfg.TaxID = "94-1234567"
	cfg.LicenseNumber = "DEN-CA-884920"
	cfg.Phone = "(555) 234-5678"
	cfg.Email = "info@apexdentalstudio.com"
	cfg.Website = "https://apexdentalstudio.example.com"
	cfg.AddressLine1 = "101 Dental Plaza, Suite 200"
	cfg.City = "San Francisco"
	cfg.StateProvince = "CA"
	cfg.PostalCode = "94105"
	cfg.NPI = "1902837461"
	cfg.DateFormat = "MM/DD/YYYY"
	cfg.Timezone = "America/Los_Angeles"
	cfg.BusinessHours = domain.DefaultBusinessHours()
	cfg.CreatedAt = now
	cfg.UpdatedAt = now

	if _, err := g.Practice.UpdatePracticeConfig(token, *cfg); err != nil {
		return fmt.Errorf("failed to seed practice config: %w", err)
	}
	summary.PracticeConfigured = true
	return nil
}

func seedProviders(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	providers := []domain.Provider{
		{
			ID:            "prov_102",
			Name:          "Dr. Marcus Vance",
			Role:          domain.RoleDentist,
			Specialty:     "Endodontics & Surgery",
			LicenseNumber: "DEN-77419",
			Email:         "m.vance@apexdentalstudio.com",
			Phone:         "555-0102",
			Color:         "#10b981",
			Pin:           "2222",
			IsActive:      true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		{
			ID:            "prov_103",
			Name:          "Elena Rostova",
			Role:          domain.RoleHygienist,
			Specialty:     "Preventive Dental Hygiene",
			LicenseNumber: "HYG-33104",
			Email:         "e.rostova@apexdentalstudio.com",
			Phone:         "555-0103",
			Color:         "#f59e0b",
			Pin:           "3333",
			IsActive:      true,
			HourlyRate:    5200,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	for _, p := range extraProviders {
		p.IsActive = true
		p.CreatedAt, p.UpdatedAt = now, now
		providers = append(providers, p)
	}

	for _, p := range providers {
		if _, err := g.Practice.SaveProvider(token, p); err != nil {
			return fmt.Errorf("failed to seed provider %s: %w", p.Name, err)
		}
		summary.ProvidersCount++
	}
	return nil
}

func seedOperatories(g *ServiceGraph, token string, now time.Time, summary *SeedSummary) error {
	operatories := []domain.Operatory{
		{
			ID:        "op_1",
			Name:      "Operatory 1",
			RoomCode:  "Room 101",
			Type:      domain.OperatoryTypeGeneral,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_2",
			Name:      "Hygiene Suite",
			RoomCode:  "Room 102",
			Type:      domain.OperatoryTypeHygiene,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "op_3",
			Name:      "Surgical Suite",
			RoomCode:  "Room 103",
			Type:      domain.OperatoryTypeSurgery,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{ID: "op_4", Name: "Operatory 2", RoomCode: "Room 104", Type: domain.OperatoryTypeGeneral, IsActive: true},
		{ID: "op_5", Name: "Operatory 3", RoomCode: "Room 105", Type: domain.OperatoryTypeGeneral, IsActive: true},
		{ID: "op_6", Name: "Hygiene Suite B", RoomCode: "Room 106", Type: domain.OperatoryTypeHygiene, IsActive: true},
		{ID: "op_7", Name: "Hygiene Suite C", RoomCode: "Room 107", Type: domain.OperatoryTypeHygiene, IsActive: true},
		{ID: "op_8", Name: "Pediatric Bay", RoomCode: "Room 108", Type: domain.OperatoryTypeGeneral, IsActive: true},
		// Closed for renovation: kept for history, hidden from booking.
		{ID: "op_9", Name: "Operatory 4", RoomCode: "Room 109", Type: domain.OperatoryTypeGeneral, IsActive: false},
		// Set up by mistake and removed again (see seedRemovals).
		{ID: "op_10", Name: "Overflow Chair", RoomCode: "Hallway", Type: domain.OperatoryTypeGeneral, IsActive: true},
	}

	for _, op := range operatories {
		op.CreatedAt, op.UpdatedAt = now, now
		if _, err := g.Practice.SaveOperatory(token, op); err != nil {
			return fmt.Errorf("failed to seed operatory %s: %w", op.Name, err)
		}
		summary.OperatoriesCount++
	}
	return nil
}

// extraProviders round out the staff of a multi-chair practice. prov_110 is a hygienist
// who has since left; seedRemovals deactivates the record once its history is recorded.
var extraProviders = []domain.Provider{
	{ID: "prov_104", Name: "Dr. Priya Raman", Role: domain.RoleDentist, Specialty: "Pediatric Dentistry", LicenseNumber: "DEN-81553", NPI: "1487261038", TaxonomyCode: "1223P0221X", Email: "p.raman@apexdentalstudio.com", Phone: "555-0104", Color: "#ec4899", Pin: "4444"},
	{ID: "prov_105", Name: "Dr. Thomas Okafor", Role: domain.RoleDentist, Specialty: "Prosthodontics", LicenseNumber: "DEN-69034", NPI: "1730592817", TaxonomyCode: "1223P0700X", Email: "t.okafor@apexdentalstudio.com", Phone: "555-0105", Color: "#8b5cf6", Pin: "5555"},
	{ID: "prov_106", Name: "James Whitfield", Role: domain.RoleHygienist, Specialty: "Periodontal Therapy", LicenseNumber: "HYG-40218", Email: "j.whitfield@apexdentalstudio.com", Phone: "555-0106", Color: "#14b8a6", Pin: "6666", HourlyRate: 5000},
	{ID: "prov_107", Name: "Aisha Karimi", Role: domain.RoleHygienist, Specialty: "Pediatric Hygiene", LicenseNumber: "HYG-42977", Email: "a.karimi@apexdentalstudio.com", Phone: "555-0107", Color: "#f97316", Pin: "7777", HourlyRate: 4800},
	{ID: "prov_108", Name: "Carla Mendes", Role: domain.RoleAssistant, Specialty: "Registered Dental Assistant", LicenseNumber: "RDA-55102", Email: "c.mendes@apexdentalstudio.com", Phone: "555-0108", Color: "#64748b", Pin: "8888", HourlyRate: 2900},
	{ID: "prov_109", Name: "Linda Foster", Role: domain.RoleStaff, Specialty: "Office Manager", Email: "l.foster@apexdentalstudio.com", Phone: "555-0109", Color: "#0ea5e9", Pin: "9999", HourlyRate: 3400},
	{ID: "prov_110", Name: "Natalie Brooks", Role: domain.RoleHygienist, Specialty: "Preventive Dental Hygiene", LicenseNumber: "HYG-31876", Email: "n.brooks@apexdentalstudio.com", Phone: "555-0110", Color: "#a3a3a3", Pin: "1010", HourlyRate: 4900},
	{ID: "prov_111", Name: "Kevin Park", Role: domain.RoleStaff, Specialty: "Front Desk Coordinator", Email: "k.park@apexdentalstudio.com", Phone: "555-0111", Color: "#22c55e", Pin: "1212", HourlyRate: 2400},
}
