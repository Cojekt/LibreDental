package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// DocumentRepository provides SQLite data access for Documents.
type DocumentRepository struct {
	db *DB
}

// NewDocumentRepository creates a new SQLite DocumentRepository.
func NewDocumentRepository(db *DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create inserts a new document record.
func (r *DocumentRepository) Create(doc *domain.Document) error {
	query := `
		INSERT INTO documents (
			id, patient_id, type, name, description, file_path, size_bytes, content_type, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.Exec(query,
		doc.ID,
		doc.PatientID,
		doc.Type,
		doc.Name,
		doc.Description,
		doc.FilePath,
		doc.SizeBytes,
		doc.ContentType,
		doc.CreatedAt,
		doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("document repo create: %w", err)
	}
	return nil
}

// GetByID retrieves a document by its ID.
func (r *DocumentRepository) GetByID(id string) (*domain.Document, error) {
	query := `
		SELECT id, patient_id, type, name, description, file_path, size_bytes, content_type, created_at, updated_at
		FROM documents
		WHERE id = ?
	`
	row := r.db.QueryRow(query, id)
	return scanDocument(row)
}

// ListByFilter retrieves documents matching the given filter.
func (r *DocumentRepository) ListByFilter(filter domain.DocumentFilter) ([]domain.Document, error) {
	query := `
		SELECT id, patient_id, type, name, description, file_path, size_bytes, content_type, created_at, updated_at
		FROM documents
		WHERE 1=1
	`
	var args []interface{}

	if filter.PatientID != nil {
		query += " AND patient_id = ?"
		args = append(args, *filter.PatientID)
	}

	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("document repo list: %w", err)
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		doc, err := scanDocumentRow(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document repo list row error: %w", err)
	}
	return docs, nil
}

// ListClinicDocuments retrieves all documents not associated with a patient.
func (r *DocumentRepository) ListClinicDocuments() ([]domain.Document, error) {
	query := `
		SELECT id, patient_id, type, name, description, file_path, size_bytes, content_type, created_at, updated_at
		FROM documents
		WHERE patient_id IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("document repo list clinic docs: %w", err)
	}
	defer rows.Close()

	var docs []domain.Document
	for rows.Next() {
		doc, err := scanDocumentRow(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, *doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("document repo list clinic docs row error: %w", err)
	}
	return docs, nil
}

// Delete removes a document record by ID.
func (r *DocumentRepository) Delete(id string) error {
	res, err := r.db.Exec("DELETE FROM documents WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("document repo delete: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("document repo delete rows affected: %w", err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func scanDocument(row rowScanner) (*domain.Document, error) {
	var doc domain.Document
	var patientID sql.NullString
	var createdAt, updatedAt time.Time
	var description sql.NullString

	err := row.Scan(
		&doc.ID,
		&patientID,
		&doc.Type,
		&doc.Name,
		&description,
		&doc.FilePath,
		&doc.SizeBytes,
		&doc.ContentType,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("document scan error: %w", err)
	}

	if patientID.Valid {
		doc.PatientID = &patientID.String
	}
	if description.Valid {
		doc.Description = description.String
	}
	doc.CreatedAt = createdAt
	doc.UpdatedAt = updatedAt

	return &doc, nil
}

func scanDocumentRow(rows *sql.Rows) (*domain.Document, error) {
	return scanDocument(rows)
}
