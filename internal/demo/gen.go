//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/LibreDental/libredental/internal/demo"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func main() {
	outputPath := "libredental.db"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}

	// Overwrite existing demo database if it exists
	if _, err := os.Stat(outputPath); err == nil {
		if err := os.Remove(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to remove existing demo db file '%s': %v\n", absPath, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Generating LibreDental™ demo database at %s...\n", absPath)

	db, err := sqlite.Open(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open SQLite database: %v\n", err)
		os.Exit(1)
	}

	summary, err := demo.SeedDatabase(db)
	_ = db.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to seed demo database: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Demo database successfully created!\n")
	fmt.Printf("   • Practice Configured : %v\n", summary.PracticeConfigured)
	fmt.Printf("   • Healthcare Providers: %d\n", summary.ProvidersCount)
	fmt.Printf("   • Operatory Rooms     : %d\n", summary.OperatoriesCount)
	fmt.Printf("   • Patients            : %d\n", summary.PatientsCount)
	fmt.Printf("   • Appointments        : %d\n", summary.AppointmentsCount)
	fmt.Printf("   • Dental Chart Records: %d\n", summary.ConditionsCount)
	fmt.Printf("   • Fee Schedule Rates  : %d\n", summary.FeeSchedulesCount)
	fmt.Printf("   • Treatment Bundles   : %d\n", summary.BundlesCount)
	fmt.Printf("   • Insurance Claims    : %d\n", summary.ClaimsCount)
	fmt.Printf("   • Patient Payments    : %d\n", summary.PaymentsCount)
	fmt.Printf("File: %s\n", absPath)
}
