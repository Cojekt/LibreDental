-- +goose Up
-- +goose StatementBegin
ALTER TABLE providers ADD COLUMN hourly_rate REAL NOT NULL DEFAULT 0.0;

CREATE TABLE IF NOT EXISTS timecards (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    clock_in DATETIME NOT NULL,
    clock_out DATETIME,
    hourly_rate REAL NOT NULL DEFAULT 0.0,
    total_hours REAL,
    total_pay REAL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS timecards;
-- +goose StatementEnd
