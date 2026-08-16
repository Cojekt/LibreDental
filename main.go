package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/LibreDental/libredental/internal/services"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	billingService := services.NewBillingService(claimRepo, paymentRepo, bundleRepo, procedureRepo, procedureRepo, chartRepo)

	documentRepo := sqlite.NewDocumentRepository(db)
	documentService := services.NewDocumentService(documentRepo, appDir)

	// Server mode configuration (used when built with: task build:server / go build -tags server).
	// LIBREDENTAL_HOST defaults to 0.0.0.0 (all LAN interfaces).
	// LIBREDENTAL_PORT defaults to 4242.
	serveHost := os.Getenv("LIBREDENTAL_HOST")
	if serveHost == "" {
		serveHost = "0.0.0.0"
	}
	servePort := 4242
	if portStr := os.Getenv("LIBREDENTAL_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			servePort = p
		} else {
			log.Printf("Warning: invalid LIBREDENTAL_PORT %q, using default %d", portStr, servePort)
		}
	}

	app := application.New(application.Options{
		Name:        "LibreDental",
		Description: "Open-Source Dental Practice Management System",
		// Server field is only active when built with -tags server.
		// In desktop mode this field is ignored by Wails.
		Server: application.ServerOptions{
			Host: serveHost,
			Port: servePort,
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

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "LibreDental",
		Width:            winWidth,
		Height:           winHeight,
		StartState:       startState,
		BackgroundColour: application.NewRGB(15, 23, 42),
		URL:              "/",
	})
	services.AttachWindow(systemSettingsService, &wailsWindowAdapter{win: win})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

type wailsWindowAdapter struct {
	win *application.WebviewWindow
}

func (w *wailsWindowAdapter) IsFullscreen() bool { return w.win.IsFullscreen() }
func (w *wailsWindowAdapter) IsMaximised() bool  { return w.win.IsMaximised() }
func (w *wailsWindowAdapter) Size() (int, int)   { return w.win.Size() }
func (w *wailsWindowAdapter) Fullscreen()        { w.win.Fullscreen() }
func (w *wailsWindowAdapter) UnFullscreen()      { w.win.UnFullscreen() }
func (w *wailsWindowAdapter) OnResize(fn func()) {
	w.win.OnWindowEvent(events.Common.WindowDidResize, func(event *application.WindowEvent) {
		fn()
	})
}
func (w *wailsWindowAdapter) OnClose(fn func()) {
	w.win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		fn()
	})
}
