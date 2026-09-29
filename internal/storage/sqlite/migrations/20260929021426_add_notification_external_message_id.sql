-- +goose Up
-- add column "external_message_id" to table: "notification_log"
ALTER TABLE `notification_log` ADD COLUMN `external_message_id` text NULL DEFAULT '';

-- +goose Down
-- reverse: add column "external_message_id" to table: "notification_log"
ALTER TABLE `notification_log` DROP COLUMN `external_message_id`;
