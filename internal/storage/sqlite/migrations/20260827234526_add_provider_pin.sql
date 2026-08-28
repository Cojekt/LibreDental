-- +goose Up
-- disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- create "new_claims" table
CREATE TABLE `new_claims` (
  `id` text NULL,
  `patient_id` text NOT NULL,
  `provider_id` text NOT NULL DEFAULT '',
  `appointment_id` text NULL DEFAULT '',
  `insurance_carrier` text NULL DEFAULT '',
  `policy_number` text NULL DEFAULT '',
  `group_number` text NULL DEFAULT '',
  `date_of_service` text NOT NULL,
  `status` text NOT NULL DEFAULT 'draft',
  `notes` text NULL DEFAULT '',
  `line_items` text NULL DEFAULT '[]',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- copy rows from old table "claims" to new temporary table "new_claims"
INSERT INTO `new_claims` (`id`, `patient_id`, `provider_id`, `appointment_id`, `insurance_carrier`, `policy_number`, `group_number`, `date_of_service`, `status`, `notes`, `line_items`, `created_at`, `updated_at`) SELECT `id`, `patient_id`, `provider_id`, `appointment_id`, `insurance_carrier`, `policy_number`, `group_number`, `date_of_service`, `status`, `notes`, `line_items`, `created_at`, `updated_at` FROM `claims`;
-- drop "claims" table after copying rows
DROP TABLE `claims`;
-- rename temporary table "new_claims" to "claims"
ALTER TABLE `new_claims` RENAME TO `claims`;
-- create index "idx_claims_patient" to table: "claims"
CREATE INDEX `idx_claims_patient` ON `claims` (`patient_id`);
-- create index "idx_claims_status" to table: "claims"
CREATE INDEX `idx_claims_status` ON `claims` (`status`);
-- create index "idx_claims_date" to table: "claims"
CREATE INDEX `idx_claims_date` ON `claims` (`date_of_service`);
-- create "new_payments" table
CREATE TABLE `new_payments` (
  `id` text NULL,
  `patient_id` text NOT NULL,
  `claim_id` text NULL DEFAULT '',
  `amount` integer NOT NULL,
  `method` text NOT NULL DEFAULT 'cash',
  `date` text NOT NULL,
  `notes` text NULL DEFAULT '',
  `created_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- copy rows from old table "payments" to new temporary table "new_payments"
INSERT INTO `new_payments` (`id`, `patient_id`, `claim_id`, `amount`, `method`, `date`, `notes`, `created_at`) SELECT `id`, `patient_id`, `claim_id`, `amount`, `method`, `date`, `notes`, `created_at` FROM `payments`;
-- drop "payments" table after copying rows
DROP TABLE `payments`;
-- rename temporary table "new_payments" to "payments"
ALTER TABLE `new_payments` RENAME TO `payments`;
-- create index "idx_payments_patient" to table: "payments"
CREATE INDEX `idx_payments_patient` ON `payments` (`patient_id`);
-- create index "idx_payments_claim" to table: "payments"
CREATE INDEX `idx_payments_claim` ON `payments` (`claim_id`);
-- create index "idx_payments_date" to table: "payments"
CREATE INDEX `idx_payments_date` ON `payments` (`date`);
-- create "new_treatment_bundles" table
CREATE TABLE `new_treatment_bundles` (
  `id` text NULL,
  `shortname` text NOT NULL,
  `name` text NOT NULL,
  `description` text NULL DEFAULT '',
  `items` text NULL DEFAULT '[]',
  `total_fee` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- copy rows from old table "treatment_bundles" to new temporary table "new_treatment_bundles"
INSERT INTO `new_treatment_bundles` (`id`, `shortname`, `name`, `description`, `items`, `total_fee`, `created_at`, `updated_at`) SELECT `id`, `shortname`, `name`, `description`, `items`, `total_fee`, `created_at`, `updated_at` FROM `treatment_bundles`;
-- drop "treatment_bundles" table after copying rows
DROP TABLE `treatment_bundles`;
-- rename temporary table "new_treatment_bundles" to "treatment_bundles"
ALTER TABLE `new_treatment_bundles` RENAME TO `treatment_bundles`;
-- create index "idx_bundles_shortname" to table: "treatment_bundles"
CREATE UNIQUE INDEX `idx_bundles_shortname` ON `treatment_bundles` (`shortname`);
-- create "new_fee_schedules" table
CREATE TABLE `new_fee_schedules` (
  `id` text NULL,
  `country_code` text NOT NULL,
  `code` text NOT NULL,
  `provider_id` text NULL DEFAULT '',
  `custom_fee` integer NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- copy rows from old table "fee_schedules" to new temporary table "new_fee_schedules"
INSERT INTO `new_fee_schedules` (`id`, `country_code`, `code`, `provider_id`, `custom_fee`, `updated_at`) SELECT `id`, `country_code`, `code`, `provider_id`, `custom_fee`, `updated_at` FROM `fee_schedules`;
-- drop "fee_schedules" table after copying rows
DROP TABLE `fee_schedules`;
-- rename temporary table "new_fee_schedules" to "fee_schedules"
ALTER TABLE `new_fee_schedules` RENAME TO `fee_schedules`;
-- create index "fee_schedules_country_code_code_provider_id" to table: "fee_schedules"
CREATE UNIQUE INDEX `fee_schedules_country_code_code_provider_id` ON `fee_schedules` (`country_code`, `code`, `provider_id`);
-- create "new_appointments" table
CREATE TABLE `new_appointments` (
  `id` text NULL,
  `patient_id` text NOT NULL,
  `provider_id` text NOT NULL,
  `operatory_id` text NOT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime NOT NULL,
  `status` text NOT NULL,
  `reason` text NULL DEFAULT '',
  `color` text NULL DEFAULT '',
  `notes` text NULL DEFAULT '',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `version` integer NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- copy rows from old table "appointments" to new temporary table "new_appointments"
INSERT INTO `new_appointments` (`id`, `patient_id`, `provider_id`, `operatory_id`, `start_time`, `end_time`, `status`, `reason`, `color`, `notes`, `created_at`, `updated_at`, `version`) SELECT `id`, `patient_id`, `provider_id`, `operatory_id`, `start_time`, `end_time`, `status`, `reason`, `color`, `notes`, `created_at`, `updated_at`, `version` FROM `appointments`;
-- drop "appointments" table after copying rows
DROP TABLE `appointments`;
-- rename temporary table "new_appointments" to "appointments"
ALTER TABLE `new_appointments` RENAME TO `appointments`;
-- create index "idx_appointments_patient" to table: "appointments"
CREATE INDEX `idx_appointments_patient` ON `appointments` (`patient_id`);
-- create index "idx_appointments_date" to table: "appointments"
CREATE INDEX `idx_appointments_date` ON `appointments` (`start_time`, `end_time`);
-- create "new_dental_conditions" table
CREATE TABLE `new_dental_conditions` (
  `id` text NULL,
  `patient_id` text NOT NULL,
  `tooth_number` integer NOT NULL,
  `surfaces` text NULL DEFAULT '[]',
  `ada_code` text NULL DEFAULT '',
  `description` text NULL DEFAULT '',
  `status` text NOT NULL,
  `fee` integer NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
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
-- create "new_practice_config" table
CREATE TABLE `new_practice_config` (
  `id` integer NULL,
  `clinic_name` text NOT NULL DEFAULT 'My Dental Clinic',
  `tagline` text NULL DEFAULT '',
  `tax_id` text NULL DEFAULT '',
  `license_number` text NULL DEFAULT '',
  `phone` text NULL DEFAULT '',
  `email` text NULL DEFAULT '',
  `website` text NULL DEFAULT '',
  `address_line1` text NULL DEFAULT '',
  `address_line2` text NULL DEFAULT '',
  `city` text NULL DEFAULT '',
  `state_province` text NULL DEFAULT '',
  `postal_code` text NULL DEFAULT '',
  `country_code` text NOT NULL,
  `currency` text NOT NULL,
  `tooth_system` text NOT NULL DEFAULT 'universal',
  `date_format` text NOT NULL DEFAULT 'YYYY-MM-DD',
  `business_hours` text NULL DEFAULT '[]',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CHECK (id = 1)
);
-- copy rows from old table "practice_config" to new temporary table "new_practice_config"
INSERT INTO `new_practice_config` (`id`, `clinic_name`, `tagline`, `tax_id`, `license_number`, `phone`, `email`, `website`, `address_line1`, `address_line2`, `city`, `state_province`, `postal_code`, `country_code`, `currency`, `tooth_system`, `date_format`, `business_hours`, `created_at`, `updated_at`) SELECT `id`, `clinic_name`, `tagline`, `tax_id`, `license_number`, `phone`, `email`, `website`, `address_line1`, `address_line2`, `city`, `state_province`, `postal_code`, `country_code`, `currency`, `tooth_system`, `date_format`, `business_hours`, `created_at`, `updated_at` FROM `practice_config`;
-- drop "practice_config" table after copying rows
DROP TABLE `practice_config`;
-- rename temporary table "new_practice_config" to "practice_config"
ALTER TABLE `new_practice_config` RENAME TO `practice_config`;
-- create "new_operatories" table
CREATE TABLE `new_operatories` (
  `id` text NULL,
  `name` text NOT NULL,
  `room_code` text NULL DEFAULT '',
  `type` text NOT NULL DEFAULT 'general',
  `is_active` integer NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- copy rows from old table "operatories" to new temporary table "new_operatories"
INSERT INTO `new_operatories` (`id`, `name`, `room_code`, `type`, `is_active`, `created_at`, `updated_at`) SELECT `id`, `name`, `room_code`, `type`, `is_active`, `created_at`, `updated_at` FROM `operatories`;
-- drop "operatories" table after copying rows
DROP TABLE `operatories`;
-- rename temporary table "new_operatories" to "operatories"
ALTER TABLE `new_operatories` RENAME TO `operatories`;
-- create "new_country_configs" table
CREATE TABLE `new_country_configs` (
  `code` text NULL,
  `name` text NOT NULL,
  `national_id_name` text NOT NULL,
  `national_id_type` text NOT NULL,
  `national_id_placeholder` text NOT NULL,
  `default_tooth_system` text NOT NULL,
  `default_currency` text NOT NULL,
  `state_province_label` text NOT NULL,
  `postal_code_label` text NOT NULL,
  `date_format` text NOT NULL,
  `is_default` integer NOT NULL DEFAULT 0,
  PRIMARY KEY (`code`)
);
-- copy rows from old table "country_configs" to new temporary table "new_country_configs"
INSERT INTO `new_country_configs` (`code`, `name`, `national_id_name`, `national_id_type`, `national_id_placeholder`, `default_tooth_system`, `default_currency`, `state_province_label`, `postal_code_label`, `date_format`, `is_default`) SELECT `code`, `name`, `national_id_name`, `national_id_type`, `national_id_placeholder`, `default_tooth_system`, `default_currency`, `state_province_label`, `postal_code_label`, `date_format`, `is_default` FROM `country_configs`;
-- drop "country_configs" table after copying rows
DROP TABLE `country_configs`;
-- rename temporary table "new_country_configs" to "country_configs"
ALTER TABLE `new_country_configs` RENAME TO `country_configs`;
-- create "new_patients" table
CREATE TABLE `new_patients` (
  `id` text NULL,
  `first_name` text NOT NULL,
  `last_name` text NOT NULL,
  `middle_name` text NULL DEFAULT '',
  `preferred_name` text NULL DEFAULT '',
  `date_of_birth` text NOT NULL,
  `sex` text NOT NULL,
  `email` text NULL DEFAULT '',
  `phone_primary` text NULL DEFAULT '',
  `phone_secondary` text NULL DEFAULT '',
  `emergency_contact_name` text NULL DEFAULT '',
  `emergency_contact_rel` text NULL DEFAULT '',
  `emergency_contact_phone` text NULL DEFAULT '',
  `guarantor_name` text NULL DEFAULT '',
  `guarantor_rel` text NULL DEFAULT '',
  `guarantor_phone` text NULL DEFAULT '',
  `insurance_carrier` text NULL DEFAULT '',
  `insurance_policy_number` text NULL DEFAULT '',
  `insurance_group_number` text NULL DEFAULT '',
  `preferred_contact_method` text NULL DEFAULT 'phone',
  `preferred_language` text NULL DEFAULT '',
  `reminder_opt_in` integer NOT NULL DEFAULT 1,
  `preferred_provider_id` text NULL DEFAULT '',
  `referral_source` text NULL DEFAULT '',
  `address_line1` text NULL DEFAULT '',
  `address_line2` text NULL DEFAULT '',
  `city` text NULL DEFAULT '',
  `state_province` text NULL DEFAULT '',
  `postal_code` text NULL DEFAULT '',
  `country_code` text NULL DEFAULT '',
  `national_id_type` text NULL DEFAULT '',
  `national_id` text NULL DEFAULT '',
  `medical_alerts` text NULL DEFAULT '[]',
  `allergies` text NULL DEFAULT '[]',
  `notes` text NULL DEFAULT '',
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `version` integer NOT NULL DEFAULT 1,
  `status` text NOT NULL DEFAULT 'active',
  PRIMARY KEY (`id`)
);
-- copy rows from old table "patients" to new temporary table "new_patients"
INSERT INTO `new_patients` (`id`, `first_name`, `last_name`, `middle_name`, `preferred_name`, `date_of_birth`, `sex`, `email`, `phone_primary`, `phone_secondary`, `emergency_contact_name`, `emergency_contact_rel`, `emergency_contact_phone`, `guarantor_name`, `guarantor_rel`, `guarantor_phone`, `insurance_carrier`, `insurance_policy_number`, `insurance_group_number`, `preferred_contact_method`, `preferred_language`, `reminder_opt_in`, `preferred_provider_id`, `referral_source`, `address_line1`, `address_line2`, `city`, `state_province`, `postal_code`, `country_code`, `national_id_type`, `national_id`, `medical_alerts`, `allergies`, `notes`, `created_at`, `updated_at`, `version`, `status`) SELECT `id`, `first_name`, `last_name`, `middle_name`, `preferred_name`, `date_of_birth`, `sex`, `email`, `phone_primary`, `phone_secondary`, `emergency_contact_name`, `emergency_contact_rel`, `emergency_contact_phone`, `guarantor_name`, `guarantor_rel`, `guarantor_phone`, `insurance_carrier`, `insurance_policy_number`, `insurance_group_number`, `preferred_contact_method`, `preferred_language`, `reminder_opt_in`, `preferred_provider_id`, `referral_source`, `address_line1`, `address_line2`, `city`, `state_province`, `postal_code`, `country_code`, `national_id_type`, `national_id`, `medical_alerts`, `allergies`, `notes`, `created_at`, `updated_at`, `version`, `status` FROM `patients`;
-- drop "patients" table after copying rows
DROP TABLE `patients`;
-- rename temporary table "new_patients" to "patients"
ALTER TABLE `new_patients` RENAME TO `patients`;
-- create index "idx_patients_name" to table: "patients"
CREATE INDEX `idx_patients_name` ON `patients` (`last_name`, `first_name`);
-- create index "idx_patients_dob" to table: "patients"
CREATE INDEX `idx_patients_dob` ON `patients` (`date_of_birth`);
-- create "new_documents" table
CREATE TABLE `new_documents` (
  `id` text NULL,
  `patient_id` text NULL,
  `type` text NOT NULL,
  `name` text NOT NULL,
  `description` text NULL,
  `file_path` text NOT NULL,
  `size_bytes` integer NOT NULL,
  `content_type` text NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- copy rows from old table "documents" to new temporary table "new_documents"
INSERT INTO `new_documents` (`id`, `patient_id`, `type`, `name`, `description`, `file_path`, `size_bytes`, `content_type`, `created_at`, `updated_at`) SELECT `id`, `patient_id`, `type`, `name`, `description`, `file_path`, `size_bytes`, `content_type`, `created_at`, `updated_at` FROM `documents`;
-- drop "documents" table after copying rows
DROP TABLE `documents`;
-- rename temporary table "new_documents" to "documents"
ALTER TABLE `new_documents` RENAME TO `documents`;
-- create index "idx_documents_patient_id" to table: "documents"
CREATE INDEX `idx_documents_patient_id` ON `documents` (`patient_id`);
-- create "new_providers" table
CREATE TABLE `new_providers` (
  `id` text NULL,
  `name` text NOT NULL,
  `role` text NOT NULL DEFAULT 'dentist',
  `specialty` text NULL DEFAULT '',
  `license_number` text NULL DEFAULT '',
  `email` text NULL DEFAULT '',
  `phone` text NULL DEFAULT '',
  `color` text NULL DEFAULT '#3b82f6',
  `pin` text NULL DEFAULT '',
  `is_active` integer NOT NULL DEFAULT 1,
  `hourly_rate` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- copy rows from old table "providers" to new temporary table "new_providers"
INSERT INTO `new_providers` (`id`, `name`, `role`, `specialty`, `license_number`, `email`, `phone`, `color`, `is_active`, `hourly_rate`, `created_at`, `updated_at`) SELECT `id`, `name`, `role`, `specialty`, `license_number`, `email`, `phone`, `color`, `is_active`, `hourly_rate`, `created_at`, `updated_at` FROM `providers`;
-- drop "providers" table after copying rows
DROP TABLE `providers`;
-- rename temporary table "new_providers" to "providers"
ALTER TABLE `new_providers` RENAME TO `providers`;
-- create "new_timecards" table
CREATE TABLE `new_timecards` (
  `id` text NULL,
  `provider_id` text NOT NULL,
  `clock_in` datetime NOT NULL,
  `clock_out` datetime NULL,
  `hourly_rate` integer NOT NULL DEFAULT 0,
  `total_minutes` integer NULL,
  `total_pay` integer NULL,
  `paid_at` datetime NULL,
  `is_manual` boolean NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- copy rows from old table "timecards" to new temporary table "new_timecards"
INSERT INTO `new_timecards` (`id`, `provider_id`, `clock_in`, `clock_out`, `hourly_rate`, `total_minutes`, `total_pay`, `paid_at`, `is_manual`, `created_at`, `updated_at`) SELECT `id`, `provider_id`, `clock_in`, `clock_out`, `hourly_rate`, `total_minutes`, `total_pay`, `paid_at`, `is_manual`, `created_at`, `updated_at` FROM `timecards`;
-- drop "timecards" table after copying rows
DROP TABLE `timecards`;
-- rename temporary table "new_timecards" to "timecards"
ALTER TABLE `new_timecards` RENAME TO `timecards`;
-- create index "idx_timecards_provider_clockin" to table: "timecards"
CREATE INDEX `idx_timecards_provider_clockin` ON `timecards` (`provider_id`, `clock_in`);
-- create index "idx_timecards_active_provider" to table: "timecards"
CREATE UNIQUE INDEX `idx_timecards_active_provider` ON `timecards` (`provider_id`) WHERE clock_out IS NULL;
-- enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;

-- +goose Down
-- reverse: create index "idx_timecards_active_provider" to table: "timecards"
DROP INDEX `idx_timecards_active_provider`;
-- reverse: create index "idx_timecards_provider_clockin" to table: "timecards"
DROP INDEX `idx_timecards_provider_clockin`;
-- reverse: create "new_timecards" table
DROP TABLE `new_timecards`;
-- reverse: create "new_providers" table
DROP TABLE `new_providers`;
-- reverse: create index "idx_documents_patient_id" to table: "documents"
DROP INDEX `idx_documents_patient_id`;
-- reverse: create "new_documents" table
DROP TABLE `new_documents`;
-- reverse: create index "idx_patients_dob" to table: "patients"
DROP INDEX `idx_patients_dob`;
-- reverse: create index "idx_patients_name" to table: "patients"
DROP INDEX `idx_patients_name`;
-- reverse: create "new_patients" table
DROP TABLE `new_patients`;
-- reverse: create "new_country_configs" table
DROP TABLE `new_country_configs`;
-- reverse: create "new_operatories" table
DROP TABLE `new_operatories`;
-- reverse: create "new_practice_config" table
DROP TABLE `new_practice_config`;
-- reverse: create index "idx_dental_conditions_tooth" to table: "dental_conditions"
DROP INDEX `idx_dental_conditions_tooth`;
-- reverse: create index "idx_dental_conditions_patient" to table: "dental_conditions"
DROP INDEX `idx_dental_conditions_patient`;
-- reverse: create "new_dental_conditions" table
DROP TABLE `new_dental_conditions`;
-- reverse: create index "idx_appointments_date" to table: "appointments"
DROP INDEX `idx_appointments_date`;
-- reverse: create index "idx_appointments_patient" to table: "appointments"
DROP INDEX `idx_appointments_patient`;
-- reverse: create "new_appointments" table
DROP TABLE `new_appointments`;
-- reverse: create index "fee_schedules_country_code_code_provider_id" to table: "fee_schedules"
DROP INDEX `fee_schedules_country_code_code_provider_id`;
-- reverse: create "new_fee_schedules" table
DROP TABLE `new_fee_schedules`;
-- reverse: create index "idx_bundles_shortname" to table: "treatment_bundles"
DROP INDEX `idx_bundles_shortname`;
-- reverse: create "new_treatment_bundles" table
DROP TABLE `new_treatment_bundles`;
-- reverse: create index "idx_payments_date" to table: "payments"
DROP INDEX `idx_payments_date`;
-- reverse: create index "idx_payments_claim" to table: "payments"
DROP INDEX `idx_payments_claim`;
-- reverse: create index "idx_payments_patient" to table: "payments"
DROP INDEX `idx_payments_patient`;
-- reverse: create "new_payments" table
DROP TABLE `new_payments`;
-- reverse: create index "idx_claims_date" to table: "claims"
DROP INDEX `idx_claims_date`;
-- reverse: create index "idx_claims_status" to table: "claims"
DROP INDEX `idx_claims_status`;
-- reverse: create index "idx_claims_patient" to table: "claims"
DROP INDEX `idx_claims_patient`;
-- reverse: create "new_claims" table
DROP TABLE `new_claims`;
