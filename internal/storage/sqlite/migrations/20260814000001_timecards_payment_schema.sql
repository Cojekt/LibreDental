-- +goose Up
-- +goose StatementBegin
ALTER TABLE timecards ADD COLUMN paid_at DATETIME;
ALTER TABLE timecards ADD COLUMN is_manual BOOLEAN NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE timecards DROP COLUMN paid_at;
ALTER TABLE timecards DROP COLUMN is_manual;
-- +goose StatementEnd
