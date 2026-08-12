-- +goose Up
-- +goose StatementBegin
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    patient_id TEXT,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_type TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients (id) ON DELETE CASCADE
);

CREATE INDEX idx_documents_patient_id ON documents (patient_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_documents_patient_id;
DROP TABLE IF EXISTS documents;
-- +goose StatementEnd
