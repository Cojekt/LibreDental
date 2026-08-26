-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,
	timestamp DATETIME NOT NULL,
	user_id TEXT NOT NULL,
	user_name TEXT NOT NULL,
	patient_id TEXT DEFAULT '',
	action TEXT NOT NULL,
	resource TEXT NOT NULL,
	resource_id TEXT DEFAULT '',
	details TEXT DEFAULT '',
	ip_address TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audit_patient ON audit_logs(patient_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs;
-- +goose StatementEnd
