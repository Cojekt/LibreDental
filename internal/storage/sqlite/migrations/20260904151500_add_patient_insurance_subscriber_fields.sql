-- +goose Up
ALTER TABLE patients ADD COLUMN insurance_is_subscriber INTEGER NOT NULL DEFAULT 0;
ALTER TABLE patients ADD COLUMN insurance_subscriber_id TEXT DEFAULT '';

-- +goose Down
ALTER TABLE patients DROP COLUMN insurance_is_subscriber;
ALTER TABLE patients DROP COLUMN insurance_subscriber_id;
