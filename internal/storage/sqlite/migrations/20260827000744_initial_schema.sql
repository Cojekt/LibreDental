-- +goose Up
-- create "claims" table
CREATE TABLE `claims` (
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
-- create index "idx_claims_patient" to table: "claims"
CREATE INDEX `idx_claims_patient` ON `claims` (`patient_id`);
-- create index "idx_claims_status" to table: "claims"
CREATE INDEX `idx_claims_status` ON `claims` (`status`);
-- create index "idx_claims_date" to table: "claims"
CREATE INDEX `idx_claims_date` ON `claims` (`date_of_service`);
-- create "payments" table
CREATE TABLE `payments` (
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
-- create index "idx_payments_patient" to table: "payments"
CREATE INDEX `idx_payments_patient` ON `payments` (`patient_id`);
-- create index "idx_payments_claim" to table: "payments"
CREATE INDEX `idx_payments_claim` ON `payments` (`claim_id`);
-- create index "idx_payments_date" to table: "payments"
CREATE INDEX `idx_payments_date` ON `payments` (`date`);
-- create "treatment_bundles" table
CREATE TABLE `treatment_bundles` (
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
-- create index "treatment_bundles_shortname" to table: "treatment_bundles"
CREATE UNIQUE INDEX `treatment_bundles_shortname` ON `treatment_bundles` (`shortname`);
-- create index "idx_bundles_shortname" to table: "treatment_bundles"
CREATE UNIQUE INDEX `idx_bundles_shortname` ON `treatment_bundles` (`shortname`);
-- create "procedure_codes" table
CREATE TABLE `procedure_codes` (
  `country_code` text NOT NULL,
  `code` text NOT NULL,
  `category` text NOT NULL,
  `description` text NOT NULL,
  `default_fee` integer NOT NULL DEFAULT 0,
  `is_active` integer NOT NULL DEFAULT 1,
  PRIMARY KEY (`country_code`, `code`)
);
-- create index "idx_procedure_codes_country" to table: "procedure_codes"
CREATE INDEX `idx_procedure_codes_country` ON `procedure_codes` (`country_code`);
-- create index "idx_procedure_codes_category" to table: "procedure_codes"
CREATE INDEX `idx_procedure_codes_category` ON `procedure_codes` (`country_code`, `category`);
-- create "fee_schedules" table
CREATE TABLE `fee_schedules` (
  `id` text NULL,
  `country_code` text NOT NULL,
  `code` text NOT NULL,
  `provider_id` text NULL DEFAULT '',
  `custom_fee` integer NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- create index "fee_schedules_country_code_code_provider_id" to table: "fee_schedules"
CREATE UNIQUE INDEX `fee_schedules_country_code_code_provider_id` ON `fee_schedules` (`country_code`, `code`, `provider_id`);
-- create index "idx_fee_schedules_lookup" to table: "fee_schedules"
CREATE INDEX `idx_fee_schedules_lookup` ON `fee_schedules` (`country_code`, `code`, `provider_id`);
-- create "appointments" table
CREATE TABLE `appointments` (
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
-- create index "idx_appointments_patient" to table: "appointments"
CREATE INDEX `idx_appointments_patient` ON `appointments` (`patient_id`);
-- create index "idx_appointments_date" to table: "appointments"
CREATE INDEX `idx_appointments_date` ON `appointments` (`start_time`, `end_time`);
-- create "dental_conditions" table
CREATE TABLE `dental_conditions` (
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
-- create index "idx_dental_conditions_patient" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_patient` ON `dental_conditions` (`patient_id`);
-- create index "idx_dental_conditions_tooth" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_tooth` ON `dental_conditions` (`patient_id`, `tooth_number`);
-- create "practice_config" table
CREATE TABLE `practice_config` (
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
-- create "operatories" table
CREATE TABLE `operatories` (
  `id` text NULL,
  `name` text NOT NULL,
  `room_code` text NULL DEFAULT '',
  `type` text NOT NULL DEFAULT 'general',
  `is_active` integer NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- create "country_configs" table
CREATE TABLE `country_configs` (
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
-- create "patients" table
CREATE TABLE `patients` (
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
-- create index "idx_patients_name" to table: "patients"
CREATE INDEX `idx_patients_name` ON `patients` (`last_name`, `first_name`);
-- create index "idx_patients_dob" to table: "patients"
CREATE INDEX `idx_patients_dob` ON `patients` (`date_of_birth`);
-- create "documents" table
CREATE TABLE `documents` (
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
-- create index "idx_documents_patient_id" to table: "documents"
CREATE INDEX `idx_documents_patient_id` ON `documents` (`patient_id`);
-- create "providers" table
CREATE TABLE `providers` (
  `id` text NULL,
  `name` text NOT NULL,
  `role` text NOT NULL DEFAULT 'dentist',
  `specialty` text NULL DEFAULT '',
  `license_number` text NULL DEFAULT '',
  `email` text NULL DEFAULT '',
  `phone` text NULL DEFAULT '',
  `color` text NULL DEFAULT '#3b82f6',
  `is_active` integer NOT NULL DEFAULT 1,
  `hourly_rate` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`)
);
-- create "timecards" table
CREATE TABLE `timecards` (
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
-- create index "idx_timecards_provider_clockin" to table: "timecards"
CREATE INDEX `idx_timecards_provider_clockin` ON `timecards` (`provider_id`, `clock_in`);
-- create index "idx_timecards_active_provider" to table: "timecards"
CREATE UNIQUE INDEX `idx_timecards_active_provider` ON `timecards` (`provider_id`) WHERE clock_out IS NULL;

-- +goose Down
-- reverse: create index "idx_timecards_active_provider" to table: "timecards"
DROP INDEX `idx_timecards_active_provider`;
-- reverse: create index "idx_timecards_provider_clockin" to table: "timecards"
DROP INDEX `idx_timecards_provider_clockin`;
-- reverse: create "timecards" table
DROP TABLE `timecards`;
-- reverse: create "providers" table
DROP TABLE `providers`;
-- reverse: create index "idx_documents_patient_id" to table: "documents"
DROP INDEX `idx_documents_patient_id`;
-- reverse: create "documents" table
DROP TABLE `documents`;
-- reverse: create index "idx_patients_dob" to table: "patients"
DROP INDEX `idx_patients_dob`;
-- reverse: create index "idx_patients_name" to table: "patients"
DROP INDEX `idx_patients_name`;
-- reverse: create "patients" table
DROP TABLE `patients`;
-- reverse: create "country_configs" table
DROP TABLE `country_configs`;
-- reverse: create "operatories" table
DROP TABLE `operatories`;
-- reverse: create "practice_config" table
DROP TABLE `practice_config`;
-- reverse: create index "idx_dental_conditions_tooth" to table: "dental_conditions"
DROP INDEX `idx_dental_conditions_tooth`;
-- reverse: create index "idx_dental_conditions_patient" to table: "dental_conditions"
DROP INDEX `idx_dental_conditions_patient`;
-- reverse: create "dental_conditions" table
DROP TABLE `dental_conditions`;
-- reverse: create index "idx_appointments_date" to table: "appointments"
DROP INDEX `idx_appointments_date`;
-- reverse: create index "idx_appointments_patient" to table: "appointments"
DROP INDEX `idx_appointments_patient`;
-- reverse: create "appointments" table
DROP TABLE `appointments`;
-- reverse: create index "idx_fee_schedules_lookup" to table: "fee_schedules"
DROP INDEX `idx_fee_schedules_lookup`;
-- reverse: create index "fee_schedules_country_code_code_provider_id" to table: "fee_schedules"
DROP INDEX `fee_schedules_country_code_code_provider_id`;
-- reverse: create "fee_schedules" table
DROP TABLE `fee_schedules`;
-- reverse: create index "idx_procedure_codes_category" to table: "procedure_codes"
DROP INDEX `idx_procedure_codes_category`;
-- reverse: create index "idx_procedure_codes_country" to table: "procedure_codes"
DROP INDEX `idx_procedure_codes_country`;
-- reverse: create "procedure_codes" table
DROP TABLE `procedure_codes`;
-- reverse: create index "idx_bundles_shortname" to table: "treatment_bundles"
DROP INDEX `idx_bundles_shortname`;
-- reverse: create index "treatment_bundles_shortname" to table: "treatment_bundles"
DROP INDEX `treatment_bundles_shortname`;
-- reverse: create "treatment_bundles" table
DROP TABLE `treatment_bundles`;
-- reverse: create index "idx_payments_date" to table: "payments"
DROP INDEX `idx_payments_date`;
-- reverse: create index "idx_payments_claim" to table: "payments"
DROP INDEX `idx_payments_claim`;
-- reverse: create index "idx_payments_patient" to table: "payments"
DROP INDEX `idx_payments_patient`;
-- reverse: create "payments" table
DROP TABLE `payments`;
-- reverse: create index "idx_claims_date" to table: "claims"
DROP INDEX `idx_claims_date`;
-- reverse: create index "idx_claims_status" to table: "claims"
DROP INDEX `idx_claims_status`;
-- reverse: create index "idx_claims_patient" to table: "claims"
DROP INDEX `idx_claims_patient`;
-- reverse: create "claims" table
DROP TABLE `claims`;
