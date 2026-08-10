package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LibreDental/libredental/internal/domain"
	"github.com/LibreDental/libredental/internal/storage"
)

// BundleRepository implements storage.TreatmentBundleRepository for SQLite.
type BundleRepository struct {
	db *DB
}

func NewBundleRepository(db *DB) *BundleRepository {
	return &BundleRepository{db: db}
}

func (r *BundleRepository) Create(ctx context.Context, b *domain.TreatmentBundle) error {
	if b.ID == "" || b.Shortname == "" || b.Name == "" {
		return fmt.Errorf("%w: ID, Shortname, and Name are required", storage.ErrInvalidInput)
	}

	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now

	if b.Items == nil {
		b.Items = []domain.BundleItemTemplate{}
	}

	itemsJSON, err := json.Marshal(b.Items)
	if err != nil {
		itemsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO treatment_bundles (id, shortname, name, description, items, total_fee, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.Shortname, b.Name, b.Description,
		string(itemsJSON), b.TotalFee,
		b.CreatedAt, b.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create treatment bundle: %w", err)
	}
	return nil
}

func (r *BundleRepository) GetByID(ctx context.Context, id string) (*domain.TreatmentBundle, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, shortname, name, description, items, total_fee, created_at, updated_at
		FROM treatment_bundles WHERE id = ?`, id)
	return scanBundle(row)
}

func (r *BundleRepository) GetByShortname(ctx context.Context, shortname string) (*domain.TreatmentBundle, error) {
	if shortname == "" {
		return nil, fmt.Errorf("%w: shortname is required", storage.ErrInvalidInput)
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, shortname, name, description, items, total_fee, created_at, updated_at
		FROM treatment_bundles WHERE shortname = ?`, shortname)
	return scanBundle(row)
}

func (r *BundleRepository) Update(ctx context.Context, b *domain.TreatmentBundle) error {
	if b.ID == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}

	b.UpdatedAt = time.Now().UTC()

	if b.Items == nil {
		b.Items = []domain.BundleItemTemplate{}
	}
	itemsJSON, err := json.Marshal(b.Items)
	if err != nil {
		itemsJSON = []byte("[]")
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE treatment_bundles SET
			shortname = ?, name = ?, description = ?, items = ?,
			total_fee = ?, updated_at = ?
		WHERE id = ?`,
		b.Shortname, b.Name, b.Description, string(itemsJSON),
		b.TotalFee, b.UpdatedAt, b.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update treatment bundle: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *BundleRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: ID is required", storage.ErrInvalidInput)
	}
	res, err := r.db.ExecContext(ctx, "DELETE FROM treatment_bundles WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete treatment bundle: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

func (r *BundleRepository) List(ctx context.Context) ([]*domain.TreatmentBundle, error) {
	query := `
		SELECT id, shortname, name, description, items, total_fee, created_at, updated_at
		FROM treatment_bundles
		ORDER BY shortname ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list treatment bundles: %w", err)
	}
	defer rows.Close()

	var bundles []*domain.TreatmentBundle
	for rows.Next() {
		b, err := scanBundle(rows)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if bundles == nil {
		bundles = []*domain.TreatmentBundle{}
	}
	return bundles, nil
}

func scanBundle(row rowScanner) (*domain.TreatmentBundle, error) {
	var b domain.TreatmentBundle
	var itemsJSON string

	err := row.Scan(
		&b.ID, &b.Shortname, &b.Name, &b.Description,
		&itemsJSON, &b.TotalFee,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, storage.ErrNotFound
	}

	if err := json.Unmarshal([]byte(itemsJSON), &b.Items); err != nil {
		b.Items = []domain.BundleItemTemplate{}
	}
	if b.Items == nil {
		b.Items = []domain.BundleItemTemplate{}
	}
	return &b, nil
}
