package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/LibreDental/libredental/pkg/services"
	"github.com/LibreDental/libredental/pkg/storage/sqlite"
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
	os.MkdirAll(appDir, 0755)

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

	systemSettingsRepo := sqlite.NewSystemSettingsRepository(db)
	systemSettingsService := services.NewSystemSettingsService(systemSettingsRepo, appDir)

	chartRepo := sqlite.NewChartRepository(db)
	chartService := services.NewChartService(chartRepo)

	app := application.New(application.Options{
		Name:        "LibreDental™",
		Description: "Open-Source Dental Practice Management System",
		Services: []application.Service{
			application.NewService(patientService),
			application.NewService(appointmentService),
			application.NewService(practiceConfigService),
			application.NewService(systemSettingsService),
			application.NewService(chartService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LibreDental™",
		Width:            1280,
		Height:           800,
		BackgroundColour: application.NewRGB(15, 23, 42),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
