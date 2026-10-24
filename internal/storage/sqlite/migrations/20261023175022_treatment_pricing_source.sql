-- +goose Up
-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_dental_conditions" table
CREATE TABLE `new_dental_conditions` (
  `id` text NOT NULL,
  `patient_id` text NOT NULL,
  `tooth_number` integer NOT NULL,
  `surfaces` text NULL DEFAULT '[]',
  `ada_code` text NULL DEFAULT '',
  `description` text NULL DEFAULT '',
  `status` text NOT NULL,
  `fee` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `insurance_provider_id` text NULL,
  `pricing_source` text NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT `1` FOREIGN KEY (`insurance_provider_id`) REFERENCES `insurance_providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT,
  CHECK (pricing_source IS NULL OR json_valid(pricing_source))
);
-- copy rows from old table "dental_conditions" to new temporary table "new_dental_conditions"
INSERT INTO `new_dental_conditions` (`id`, `patient_id`, `tooth_number`, `surfaces`, `ada_code`, `description`, `status`, `fee`, `created_at`, `updated_at`) SELECT `id`, `patient_id`, `tooth_number`, `surfaces`, `ada_code`, `description`, `status`, `fee`, `created_at`, `updated_at` FROM `dental_conditions`;
-- drop "dental_conditions" table after copying rows
DROP TABLE `dental_conditions`;
-- rename temporary table "new_dental_conditions" to "dental_conditions"
ALTER TABLE `new_dental_conditions` RENAME TO `dental_conditions`;
-- create index "idx_dental_conditions_patient" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_patient` ON `dental_conditions` (`patient_id`);
-- create index "idx_dental_conditions_tooth" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_tooth` ON `dental_conditions` (`patient_id`, `tooth_number`);
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;

-- +goose Down
-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Drop index "insurance_providers_normalized_name" from table: "insurance_providers"
DROP INDEX `insurance_providers_normalized_name`;
-- Create index "insurance_providers_normalized_name" to table: "insurance_providers"
CREATE UNIQUE INDEX `insurance_providers_normalized_name` ON `insurance_providers` (`normalized_name`);
-- Create "new_dental_conditions" table
CREATE TABLE `new_dental_conditions` (
  `id` text NOT NULL,
  `patient_id` text NOT NULL,
  `tooth_number` integer NOT NULL,
  `surfaces` text NULL DEFAULT '[]',
  `ada_code` text NULL DEFAULT '',
  `description` text NULL DEFAULT '',
  `status` text NOT NULL,
  `fee` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- Copy rows from old table "dental_conditions" to new temporary table "new_dental_conditions"
INSERT INTO `new_dental_conditions` (`id`, `patient_id`, `tooth_number`, `surfaces`, `ada_code`, `description`, `status`, `fee`, `created_at`, `updated_at`) SELECT `id`, `patient_id`, `tooth_number`, `surfaces`, `ada_code`, `description`, `status`, `fee`, `created_at`, `updated_at` FROM `dental_conditions`;
-- Drop "dental_conditions" table after copying rows
DROP TABLE `dental_conditions`;
-- Rename temporary table "new_dental_conditions" to "dental_conditions"
ALTER TABLE `new_dental_conditions` RENAME TO `dental_conditions`;
-- Create index "idx_dental_conditions_patient" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_patient` ON `dental_conditions` (`patient_id`);
-- Create index "idx_dental_conditions_tooth" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_tooth` ON `dental_conditions` (`patient_id`, `tooth_number`);
-- Drop index "reminder_rules_offset_minutes_channel" from table: "reminder_rules"
DROP INDEX `reminder_rules_offset_minutes_channel`;
-- Create index "reminder_rules_offset_minutes_channel" to table: "reminder_rules"
CREATE UNIQUE INDEX `reminder_rules_offset_minutes_channel` ON `reminder_rules` (`offset_minutes`, `channel`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
