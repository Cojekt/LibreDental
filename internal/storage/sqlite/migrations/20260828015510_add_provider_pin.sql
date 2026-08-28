-- +goose Up
-- add column "pin" to table: "providers"
ALTER TABLE `providers` ADD COLUMN `pin` text NULL DEFAULT '';

-- +goose Down
-- reverse: add column "pin" to table: "providers"
ALTER TABLE `providers` DROP COLUMN `pin`;
