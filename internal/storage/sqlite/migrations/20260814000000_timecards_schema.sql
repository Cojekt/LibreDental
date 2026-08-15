-- +goose Up
-- +goose StatementBegin
ALTER TABLE providers ADD COLUMN hourly_rate INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS timecards (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    clock_in DATETIME NOT NULL,
    clock_out DATETIME,
    hourly_rate INTEGER NOT NULL DEFAULT 0,
    total_minutes INTEGER,
    total_pay INTEGER,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (provider_id) REFERENCES providers (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_timecards_provider_clockin ON timecards (provider_id, clock_in);
CREATE UNIQUE INDEX IF NOT EXISTS idx_timecards_active_provider ON timecards (provider_id) WHERE clock_out IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_timecards_active_provider;
DROP INDEX IF EXISTS idx_timecards_provider_clockin;
DROP TABLE IF EXISTS timecards;
-- +goose StatementEnd

