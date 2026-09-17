package demo

import (
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

// ServiceGraph bundles the same Go services the Wails frontend binds to, wired up
// exactly as main.go wires them. The demo generator drives these services directly
// instead of touching the database or its repositories, so demo data can never
// drift from how the real application creates and validates records.
type ServiceGraph struct {
	Patient     *services.PatientService
	Appointment *services.AppointmentService
	Practice    *services.PracticeConfigService
	Chart       *services.ChartService
	Billing     *services.BillingService
	Document    *services.DocumentService
	Audit       *services.AuditService
}

// BuildServiceGraph constructs the full service graph against the given main and
// audit databases and application directory, mirroring the construction order in
// main.go (repos -> AuditService -> every other service).
func BuildServiceGraph(db *sqlite.DB, auditDb *sqlite.DB, appDir string) *ServiceGraph {
	auditRepo := sqlite.NewAuditRepository(auditDb)
	practiceConfigRepo := sqlite.NewPracticeConfigRepository(db)
	auditService := services.NewAuditService(auditRepo, practiceConfigRepo)

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo, auditService)

	appointmentRepo := sqlite.NewAppointmentRepository(db)
	appointmentService := services.NewAppointmentService(appointmentRepo, auditService)

	practiceConfigService := services.NewPracticeConfigService(practiceConfigRepo, auditService)

	chartRepo := sqlite.NewChartRepository(db)
	chartService := services.NewChartService(chartRepo, auditService)

	claimRepo := sqlite.NewClaimRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procedureRepo := sqlite.NewProcedureRepository(db)
	secretsService := services.NewSecretsService()
	billingService := services.NewBillingService(claimRepo, paymentRepo, bundleRepo, procedureRepo, procedureRepo, chartRepo, secretsService, auditService)

	documentRepo := sqlite.NewDocumentRepository(db)
	documentService := services.NewDocumentService(documentRepo, appDir, auditService)

	return &ServiceGraph{
		Patient:     patientService,
		Appointment: appointmentService,
		Practice:    practiceConfigService,
		Chart:       chartService,
		Billing:     billingService,
		Document:    documentService,
		Audit:       auditService,
	}
}
