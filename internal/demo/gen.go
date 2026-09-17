//go:build ignore

package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/LibreDental/libredental/internal/demo"
	"github.com/LibreDental/libredental/internal/storage/sqlite"
)

func main() {
	outputPath := "libredental-demo-data.zip"
	if len(os.Args) > 1 {
		outputPath = os.Args[1]
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		absPath = outputPath
	}

	fmt.Printf("Generating LibreDental™ demo save archive at %s...\n", absPath)

	tempDir, err := os.MkdirTemp("", "libredental_demo_*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir) // clean up

	// tempDir doubles as the save folder (appDir): the main DB, audit DB, documents,
	// and app settings are all written directly into it by the same services the real
	// app uses, so zipping tempDir's contents produces a complete drop-in save folder.
	dbPath := filepath.Join(tempDir, "libredental.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open SQLite database: %v\n", err)
		os.Exit(1)
	}

	auditDbPath := filepath.Join(tempDir, "audit.db")
	auditDb, err := sqlite.OpenAudit(auditDbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to open audit SQLite database: %v\n", err)
		os.Exit(1)
	}

	demoDataDir := filepath.Join(".", "internal", "demo", "data")
	summary, err := demo.SeedDatabase(db, auditDb, tempDir, demoDataDir)
	_ = db.Close()
	_ = auditDb.Close()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to seed demo database: %v\n", err)
		os.Exit(1)
	}

	// Zip the temp directory contents under a "LibreDental/" root folder, so
	// extracting the archive reproduces the real save folder 1:1 (matching
	// os.UserConfigDir()/LibreDental) rather than dumping files loose.
	if err := zipDirectory(tempDir, absPath, "LibreDental"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to create zip archive: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Demo save archive successfully created!\n")
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
	fmt.Printf("   • Documents Saved     : %d\n", summary.DocumentsCount)
	fmt.Printf("File: %s\n", absPath)
}

func zipDirectory(sourceDir, targetZipFile, rootFolderName string) error {
	zipFile, err := os.Create(targetZipFile)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path for zip entry
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip root dir
		if relPath == "." {
			return nil
		}

		// Always use forward slashes in zip
		relPath = strings.ReplaceAll(relPath, string(os.PathSeparator), "/")
		relPath = rootFolderName + "/" + relPath

		if info.IsDir() {
			relPath += "/"
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
	if err != nil {
		return err
	}

	if err := archive.Close(); err != nil {
		return err
	}

	return zipFile.Close()
}
