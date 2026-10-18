-- +goose Up
-- add column "timezone" to table: "practice_config"
ALTER TABLE `practice_config` ADD COLUMN `timezone` text NULL DEFAULT '';
-- add column "reminder_kind" to table: "notification_log"
ALTER TABLE `notification_log` ADD COLUMN `reminder_kind` text NULL;
-- add column "appointment_start" to table: "notification_log"
ALTER TABLE `notification_log` ADD COLUMN `appointment_start` text NULL;
-- create index "idx_notification_log_reminder" to table: "notification_log"
CREATE UNIQUE INDEX `idx_notification_log_reminder` ON `notification_log` (`appointment_id`, `appointment_start`, `reminder_kind`, `channel`) WHERE reminder_kind IS NOT NULL;
-- create "reminder_settings" table
CREATE TABLE `reminder_settings` (
  `id` integer NOT NULL,
  `enabled` integer NOT NULL DEFAULT 0,
  `sending_hours_start` text NOT NULL DEFAULT '08:00',
  `sending_hours_end` text NOT NULL DEFAULT '20:00',
  `enabled_at` text NULL,
  `enabled_by` text NULL,
  `updated_at` text NOT NULL,
  PRIMARY KEY (`id`),
  CHECK (id = 1)
);
-- create "reminder_rules" table
CREATE TABLE `reminder_rules` (
  `id` text NOT NULL,
  `offset_minutes` integer NOT NULL,
  `channel` text NOT NULL,
  `provider_name` text NOT NULL,
  `subject_template` text NOT NULL DEFAULT '',
  `body_template` text NOT NULL,
  `enabled` integer NOT NULL DEFAULT 1,
  `created_at` text NOT NULL,
  `updated_at` text NOT NULL,
  PRIMARY KEY (`id`)
);
-- create index "reminder_rules_offset_minutes_channel" to table: "reminder_rules"
CREATE UNIQUE INDEX `reminder_rules_offset_minutes_channel` ON `reminder_rules` (`offset_minutes`, `channel`);

-- +goose Down
-- reverse: create index "reminder_rules_offset_minutes_channel" to table: "reminder_rules"
DROP INDEX `reminder_rules_offset_minutes_channel`;
-- reverse: create "reminder_rules" table
DROP TABLE `reminder_rules`;
-- reverse: create "reminder_settings" table
DROP TABLE `reminder_settings`;
-- reverse: create index "idx_notification_log_reminder" to table: "notification_log"
DROP INDEX `idx_notification_log_reminder`;
-- reverse: add column "appointment_start" to table: "notification_log"
ALTER TABLE `notification_log` DROP COLUMN `appointment_start`;
-- reverse: add column "reminder_kind" to table: "notification_log"
ALTER TABLE `notification_log` DROP COLUMN `reminder_kind`;
-- reverse: add column "timezone" to table: "practice_config"
ALTER TABLE `practice_config` DROP COLUMN `timezone`;
