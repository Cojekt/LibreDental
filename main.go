package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/LibreDental/libredental/internal/app"
	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Initialize local SQLite database
	dataDir, err := os.UserConfigDir()
	if err != nil {
		dataDir = "."
	}
	appDir := filepath.Join(dataDir, "LibreDental")
	os.MkdirAll(appDir, 0o755)

	dbPath := filepath.Join(appDir, "libredental.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite database: %v", err)
	}
	defer db.Close()

	patientRepo := sqlite.NewPatientRepository(db)
	patientService := services.NewPatientService(patientRepo)

	appointmentRepo := sqlite.NewAppointmentRepository(db)
	appointmentService := services.NewAppointmentService(appointmentRepo)

	practiceConfigRepo := sqlite.NewPracticeConfigRepository(db)
	practiceConfigService := services.NewPracticeConfigService(practiceConfigRepo)

	timecardRepo := sqlite.NewTimecardRepository(db)
	timecardService := services.NewTimecardService(timecardRepo, practiceConfigRepo)

	systemSettingsService := services.NewSystemSettingsService(appDir)

	chartRepo := sqlite.NewChartRepository(db)
	chartService := services.NewChartService(chartRepo)

	claimRepo := sqlite.NewClaimRepository(db)
	paymentRepo := sqlite.NewPaymentRepository(db)
	bundleRepo := sqlite.NewBundleRepository(db)
	procedureRepo := sqlite.NewProcedureRepository(db)

	secretsService := services.NewSecretsService()
	billingService := services.NewBillingService(claimRepo, paymentRepo, bundleRepo, procedureRepo, procedureRepo, chartRepo, secretsService)

	documentRepo := sqlite.NewDocumentRepository(db)
	documentService := services.NewDocumentService(documentRepo, appDir)

	serverCfg := app.LoadServerConfig()

	wailsApp := application.New(application.Options{
		Name:        "LibreDental",
		Description: "Open-Source Dental Practice Management System",
		// Server field is only active when built with -tags server.
		// In desktop mode this field is ignored by Wails.
		Server: application.ServerOptions{
			Host: serverCfg.Host,
			Port: serverCfg.Port,
		},
		Services: []application.Service{
			application.NewService(patientService),
			application.NewService(appointmentService),
			application.NewService(practiceConfigService),
			application.NewService(systemSettingsService),
			application.NewService(chartService),
			application.NewService(billingService),
			application.NewService(documentService),
			application.NewService(timecardService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	winWidth, winHeight, _ := systemSettingsService.GetWindowSize()
	winMode, _ := systemSettingsService.GetWindowMode()

	startState := application.WindowStateNormal
	if winMode == "fullscreen" {
		startState = application.WindowStateFullscreen
	}

	win := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LibreDental",
		Width:            winWidth,
		Height:           winHeight,
		StartState:       startState,
		BackgroundColour: application.NewRGB(15, 23, 42),
		URL:              "/",
	})
	if win != nil {
		services.AttachWindow(systemSettingsService, app.NewWailsWindowAdapter(win))
	}

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
