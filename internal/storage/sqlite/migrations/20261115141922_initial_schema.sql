-- +goose Up
-- create "claims" table
CREATE TABLE `claims` (`id` text NOT NULL, `patient_id` text NOT NULL, `provider_id` text NULL, `appointment_id` text NULL, `insurance_carrier` text NULL DEFAULT '', `policy_number` text NULL DEFAULT '', `group_number` text NULL DEFAULT '', `date_of_service` text NOT NULL, `status` text NOT NULL DEFAULT 'draft', `notes` text NULL DEFAULT '', `line_items` text NOT NULL DEFAULT '[]', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `payer_id` text NULL DEFAULT '', PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`appointment_id`) REFERENCES `appointments` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT `1` FOREIGN KEY (`provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT `2` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (date(date_of_service) IS date_of_service), CHECK (json_valid(line_items)), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "idx_claims_patient" to table: "claims"
CREATE INDEX `idx_claims_patient` ON `claims` (`patient_id`);
-- create index "idx_claims_provider" to table: "claims"
CREATE INDEX `idx_claims_provider` ON `claims` (`provider_id`);
-- create index "idx_claims_appointment" to table: "claims"
CREATE INDEX `idx_claims_appointment` ON `claims` (`appointment_id`);
-- create index "idx_claims_status" to table: "claims"
CREATE INDEX `idx_claims_status` ON `claims` (`status`);
-- create index "idx_claims_date" to table: "claims"
CREATE INDEX `idx_claims_date` ON `claims` (`date_of_service`);
-- create "payments" table
CREATE TABLE `payments` (`id` text NOT NULL, `patient_id` text NOT NULL, `claim_id` text NULL, `amount` integer NOT NULL, `method` text NOT NULL DEFAULT 'cash', `date` text NOT NULL, `notes` text NULL DEFAULT '', `created_at` integer NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`claim_id`) REFERENCES `claims` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT `1` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (date(date) IS date), CHECK (typeof(created_at) = 'integer'));
-- create index "idx_payments_patient" to table: "payments"
CREATE INDEX `idx_payments_patient` ON `payments` (`patient_id`);
-- create index "idx_payments_claim" to table: "payments"
CREATE INDEX `idx_payments_claim` ON `payments` (`claim_id`);
-- create index "idx_payments_date" to table: "payments"
CREATE INDEX `idx_payments_date` ON `payments` (`date`);
-- create "treatment_bundles" table
CREATE TABLE `treatment_bundles` (`id` text NOT NULL, `shortname` text NOT NULL, `name` text NOT NULL, `description` text NULL DEFAULT '', `items` text NOT NULL DEFAULT '[]', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CHECK (json_valid(items)), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "idx_bundles_shortname" to table: "treatment_bundles"
CREATE UNIQUE INDEX `idx_bundles_shortname` ON `treatment_bundles` (`shortname`);
-- create "procedure_codes" table
CREATE TABLE `procedure_codes` (`country_code` text NOT NULL, `code` text NOT NULL, `category` text NOT NULL, `description` text NOT NULL, `default_fee` integer NOT NULL DEFAULT 0, `is_active` integer NOT NULL DEFAULT 1, PRIMARY KEY (`country_code`, `code`));
-- create index "idx_procedure_codes_category" to table: "procedure_codes"
CREATE INDEX `idx_procedure_codes_category` ON `procedure_codes` (`country_code`, `category`);
-- create "fee_schedules" table
CREATE TABLE `fee_schedules` (`id` text NOT NULL, `country_code` text NOT NULL, `code` text NOT NULL, `provider_id` text NULL, `custom_fee` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`country_code`, `code`) REFERENCES `procedure_codes` (`country_code`, `code`) ON UPDATE NO ACTION ON DELETE CASCADE, CONSTRAINT `1` FOREIGN KEY (`provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE, CHECK (typeof(updated_at) = 'integer'));
-- create index "idx_fee_schedules_provider" to table: "fee_schedules"
CREATE UNIQUE INDEX `idx_fee_schedules_provider` ON `fee_schedules` (`country_code`, `code`, `provider_id`) WHERE provider_id IS NOT NULL;
-- create index "idx_fee_schedules_practice" to table: "fee_schedules"
CREATE UNIQUE INDEX `idx_fee_schedules_practice` ON `fee_schedules` (`country_code`, `code`) WHERE provider_id IS NULL;
-- create "insurance_providers" table
CREATE TABLE `insurance_providers` (`id` text NOT NULL, `name` text NOT NULL, `normalized_name` text NOT NULL, `archived` integer NOT NULL DEFAULT 0, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CHECK (length(normalized_name) > 0), CHECK (archived IN (0, 1)), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "insurance_providers_normalized_name" to table: "insurance_providers"
CREATE UNIQUE INDEX `insurance_providers_normalized_name` ON `insurance_providers` (`normalized_name`);
-- create "insurance_fee_overrides" table
CREATE TABLE `insurance_fee_overrides` (`provider_id` text NOT NULL, `country_code` text NOT NULL, `code` text NOT NULL, `fee` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`provider_id`, `country_code`, `code`), CONSTRAINT `0` FOREIGN KEY (`country_code`, `code`) REFERENCES `procedure_codes` (`country_code`, `code`) ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT `1` FOREIGN KEY (`provider_id`) REFERENCES `insurance_providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (fee >= 0 AND fee <= 9007199254740991), CHECK (typeof(updated_at) = 'integer'));
-- create "appointments" table
CREATE TABLE `appointments` (`id` text NOT NULL, `patient_id` text NOT NULL, `provider_id` text NOT NULL, `operatory_id` text NOT NULL, `start_time` integer NOT NULL, `end_time` integer NOT NULL, `status` text NOT NULL, `reason` text NULL DEFAULT '', `color` text NULL DEFAULT '', `notes` text NULL DEFAULT '', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `version` integer NOT NULL DEFAULT 1, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`operatory_id`) REFERENCES `operatories` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT `1` FOREIGN KEY (`provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT `2` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (typeof(start_time) = 'integer'), CHECK (typeof(end_time) = 'integer'), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "idx_appointments_patient" to table: "appointments"
CREATE INDEX `idx_appointments_patient` ON `appointments` (`patient_id`);
-- create index "idx_appointments_provider" to table: "appointments"
CREATE INDEX `idx_appointments_provider` ON `appointments` (`provider_id`, `start_time`);
-- create index "idx_appointments_operatory" to table: "appointments"
CREATE INDEX `idx_appointments_operatory` ON `appointments` (`operatory_id`, `start_time`);
-- create index "idx_appointments_date" to table: "appointments"
CREATE INDEX `idx_appointments_date` ON `appointments` (`start_time`, `end_time`);
-- create "dental_conditions" table
CREATE TABLE `dental_conditions` (`id` text NOT NULL, `patient_id` text NOT NULL, `tooth_number` integer NOT NULL, `surfaces` text NULL DEFAULT '[]', `ada_code` text NULL DEFAULT '', `description` text NULL DEFAULT '', `status` text NOT NULL, `fee` integer NOT NULL DEFAULT 0, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `insurance_provider_id` text NULL, `pricing_source` text NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CONSTRAINT `1` FOREIGN KEY (`insurance_provider_id`) REFERENCES `insurance_providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'), CHECK (pricing_source IS NULL OR json_valid(pricing_source)));
-- create index "idx_dental_conditions_patient" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_patient` ON `dental_conditions` (`patient_id`);
-- create index "idx_dental_conditions_tooth" to table: "dental_conditions"
CREATE INDEX `idx_dental_conditions_tooth` ON `dental_conditions` (`patient_id`, `tooth_number`);
-- create "practice_config" table
CREATE TABLE `practice_config` (`id` integer NOT NULL, `clinic_name` text NOT NULL DEFAULT 'My Dental Clinic', `tagline` text NULL DEFAULT '', `tax_id` text NULL DEFAULT '', `license_number` text NULL DEFAULT '', `phone` text NULL DEFAULT '', `email` text NULL DEFAULT '', `website` text NULL DEFAULT '', `address_line1` text NULL DEFAULT '', `address_line2` text NULL DEFAULT '', `city` text NULL DEFAULT '', `state_province` text NULL DEFAULT '', `postal_code` text NULL DEFAULT '', `country_code` text NOT NULL, `currency` text NOT NULL, `tooth_system` text NOT NULL DEFAULT 'universal', `date_format` text NOT NULL DEFAULT 'YYYY-MM-DD', `business_hours` text NULL DEFAULT '[]', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `npi` text NULL DEFAULT '', `timezone` text NULL DEFAULT '', PRIMARY KEY (`id`), CHECK (id = 1), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create "operatories" table
CREATE TABLE `operatories` (`id` text NOT NULL, `name` text NOT NULL, `room_code` text NULL DEFAULT '', `type` text NOT NULL DEFAULT 'general', `is_active` integer NOT NULL DEFAULT 1, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create "country_configs" table
CREATE TABLE `country_configs` (`code` text NOT NULL, `name` text NOT NULL, `national_id_name` text NOT NULL, `national_id_type` text NOT NULL, `national_id_placeholder` text NOT NULL, `default_tooth_system` text NOT NULL, `default_currency` text NOT NULL, `state_province_label` text NOT NULL, `postal_code_label` text NOT NULL, `date_format` text NOT NULL, `is_default` integer NOT NULL DEFAULT 0, PRIMARY KEY (`code`));
-- create "program_bridges" table
CREATE TABLE `program_bridges` (`name` text NOT NULL, `enabled` integer NOT NULL DEFAULT 0, `path` text NOT NULL DEFAULT '', `args` text NOT NULL DEFAULT '', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`name`), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create "notification_log" table
CREATE TABLE `notification_log` (`id` text NOT NULL, `patient_id` text NOT NULL, `appointment_id` text NULL, `channel` text NOT NULL, `provider_name` text NOT NULL, `external_message_id` text NULL DEFAULT '', `recipient` text NOT NULL, `subject` text NULL DEFAULT '', `body` text NOT NULL DEFAULT '', `status` text NOT NULL, `error_message` text NULL DEFAULT '', `sent_at` integer NOT NULL, `reminder_kind` text NULL, `appointment_start` integer NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`appointment_id`) REFERENCES `appointments` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL, CONSTRAINT `1` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (typeof(sent_at) = 'integer'), CHECK (appointment_start IS NULL OR typeof(appointment_start) = 'integer'));
-- create index "idx_notification_log_patient" to table: "notification_log"
CREATE INDEX `idx_notification_log_patient` ON `notification_log` (`patient_id`);
-- create index "idx_notification_log_appointment" to table: "notification_log"
CREATE INDEX `idx_notification_log_appointment` ON `notification_log` (`appointment_id`);
-- create index "idx_notification_log_sent_at" to table: "notification_log"
CREATE INDEX `idx_notification_log_sent_at` ON `notification_log` (`sent_at`);
-- create index "idx_notification_log_reminder" to table: "notification_log"
CREATE UNIQUE INDEX `idx_notification_log_reminder` ON `notification_log` (`appointment_id`, `appointment_start`, `reminder_kind`, `channel`) WHERE reminder_kind IS NOT NULL AND status <> 'skipped';
-- create "reminder_settings" table
CREATE TABLE `reminder_settings` (`id` integer NOT NULL, `enabled` integer NOT NULL DEFAULT 0, `sending_hours_start` text NOT NULL DEFAULT '08:00', `sending_hours_end` text NOT NULL DEFAULT '20:00', `enabled_at` integer NULL, `enabled_by` text NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CHECK (id = 1), CHECK (enabled_at IS NULL OR typeof(enabled_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create "reminder_rules" table
CREATE TABLE `reminder_rules` (`id` text NOT NULL, `offset_minutes` integer NOT NULL, `channel` text NOT NULL, `provider_name` text NOT NULL, `subject_template` text NOT NULL DEFAULT '', `body_template` text NOT NULL, `enabled` integer NOT NULL DEFAULT 1, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "reminder_rules_offset_minutes_channel" to table: "reminder_rules"
CREATE UNIQUE INDEX `reminder_rules_offset_minutes_channel` ON `reminder_rules` (`offset_minutes`, `channel`);
-- create "patients" table
CREATE TABLE `patients` (`id` text NOT NULL, `first_name` text NOT NULL, `last_name` text NOT NULL, `preferred_name` text NULL DEFAULT '', `date_of_birth` text NOT NULL, `sex` text NOT NULL, `email` text NULL DEFAULT '', `phone_primary` text NULL DEFAULT '', `phone_secondary` text NULL DEFAULT '', `emergency_contact_name` text NULL DEFAULT '', `emergency_contact_rel` text NULL DEFAULT '', `emergency_contact_phone` text NULL DEFAULT '', `guarantor_name` text NULL DEFAULT '', `guarantor_rel` text NULL DEFAULT '', `guarantor_phone` text NULL DEFAULT '', `insurance_carrier` text NULL DEFAULT '', `insurance_policy_number` text NULL DEFAULT '', `insurance_group_number` text NULL DEFAULT '', `insurance_is_subscriber` integer NOT NULL DEFAULT 0, `preferred_contact_method` text NULL DEFAULT 'phone', `preferred_language` text NULL DEFAULT '', `reminder_opt_in` integer NOT NULL DEFAULT 1, `preferred_provider_id` text NULL, `referral_source` text NULL DEFAULT '', `address_line1` text NULL DEFAULT '', `address_line2` text NULL DEFAULT '', `city` text NULL DEFAULT '', `state_province` text NULL DEFAULT '', `postal_code` text NULL DEFAULT '', `country_code` text NULL DEFAULT '', `national_id_type` text NULL DEFAULT '', `national_id` text NULL DEFAULT '', `medical_alerts` text NULL DEFAULT '[]', `allergies` text NULL DEFAULT '[]', `notes` text NULL DEFAULT '', `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `version` integer NOT NULL DEFAULT 1, `status` text NOT NULL DEFAULT 'active', `insurance_payer_id` text NULL DEFAULT '', `insurance_subscriber_first_name` text NULL DEFAULT '', `insurance_subscriber_last_name` text NULL DEFAULT '', `insurance_subscriber_dob` text NULL DEFAULT '', `insurance_subscriber_sex` text NULL DEFAULT '', `insurance_subscriber_relationship` text NULL DEFAULT '', `insurance_subscriber_address_line1` text NULL DEFAULT '', `insurance_subscriber_address_line2` text NULL DEFAULT '', `insurance_subscriber_city` text NULL DEFAULT '', `insurance_subscriber_state_province` text NULL DEFAULT '', `insurance_subscriber_postal_code` text NULL DEFAULT '', PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`preferred_provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL, CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'), CHECK (date(date_of_birth) IS date_of_birth));
-- create index "idx_patients_name" to table: "patients"
CREATE INDEX `idx_patients_name` ON `patients` (`last_name`, `first_name`);
-- create index "idx_patients_dob" to table: "patients"
CREATE INDEX `idx_patients_dob` ON `patients` (`date_of_birth`);
-- create "documents" table
CREATE TABLE `documents` (`id` text NOT NULL, `patient_id` text NULL, `type` text NOT NULL, `name` text NOT NULL, `description` text NULL, `file_path` text NOT NULL, `size_bytes` integer NOT NULL, `content_type` text NOT NULL, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT, CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create index "idx_documents_patient_id" to table: "documents"
CREATE INDEX `idx_documents_patient_id` ON `documents` (`patient_id`);
-- create "providers" table
CREATE TABLE `providers` (`id` text NOT NULL, `name` text NOT NULL, `role` text NOT NULL DEFAULT 'dentist', `specialty` text NULL DEFAULT '', `license_number` text NULL DEFAULT '', `email` text NULL DEFAULT '', `phone` text NULL DEFAULT '', `color` text NULL DEFAULT '#3b82f6', `pin` text NULL DEFAULT '', `is_active` integer NOT NULL DEFAULT 1, `hourly_rate` integer NOT NULL DEFAULT 0, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, `npi` text NULL DEFAULT '', `taxonomy_code` text NULL DEFAULT '', PRIMARY KEY (`id`), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
-- create "timecards" table
CREATE TABLE `timecards` (`id` text NOT NULL, `provider_id` text NOT NULL, `clock_in` integer NOT NULL, `clock_out` integer NULL, `hourly_rate` integer NOT NULL DEFAULT 0, `total_minutes` integer NULL, `total_pay` integer NULL AS ((total_minutes * hourly_rate + 30) / 60) VIRTUAL, `paid_at` integer NULL, `is_manual` boolean NOT NULL DEFAULT 0, `created_at` integer NOT NULL, `updated_at` integer NOT NULL, PRIMARY KEY (`id`), CONSTRAINT `0` FOREIGN KEY (`provider_id`) REFERENCES `providers` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION, CHECK (typeof(clock_in) = 'integer'), CHECK (clock_out IS NULL OR typeof(clock_out) = 'integer'), CHECK (paid_at IS NULL OR typeof(paid_at) = 'integer'), CHECK (typeof(created_at) = 'integer'), CHECK (typeof(updated_at) = 'integer'));
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
-- reverse: create index "reminder_rules_offset_minutes_channel" to table: "reminder_rules"
DROP INDEX `reminder_rules_offset_minutes_channel`;
-- reverse: create "reminder_rules" table
DROP TABLE `reminder_rules`;
-- reverse: create "reminder_settings" table
DROP TABLE `reminder_settings`;
-- reverse: create index "idx_notification_log_reminder" to table: "notification_log"
DROP INDEX `idx_notification_log_reminder`;
-- reverse: create index "idx_notification_log_sent_at" to table: "notification_log"
DROP INDEX `idx_notification_log_sent_at`;
-- reverse: create index "idx_notification_log_appointment" to table: "notification_log"
DROP INDEX `idx_notification_log_appointment`;
-- reverse: create index "idx_notification_log_patient" to table: "notification_log"
DROP INDEX `idx_notification_log_patient`;
-- reverse: create "notification_log" table
DROP TABLE `notification_log`;
-- reverse: create "program_bridges" table
DROP TABLE `program_bridges`;
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
-- reverse: create index "idx_appointments_operatory" to table: "appointments"
DROP INDEX `idx_appointments_operatory`;
-- reverse: create index "idx_appointments_provider" to table: "appointments"
DROP INDEX `idx_appointments_provider`;
-- reverse: create index "idx_appointments_patient" to table: "appointments"
DROP INDEX `idx_appointments_patient`;
-- reverse: create "appointments" table
DROP TABLE `appointments`;
-- reverse: create "insurance_fee_overrides" table
DROP TABLE `insurance_fee_overrides`;
-- reverse: create index "insurance_providers_normalized_name" to table: "insurance_providers"
DROP INDEX `insurance_providers_normalized_name`;
-- reverse: create "insurance_providers" table
DROP TABLE `insurance_providers`;
-- reverse: create index "idx_fee_schedules_practice" to table: "fee_schedules"
DROP INDEX `idx_fee_schedules_practice`;
-- reverse: create index "idx_fee_schedules_provider" to table: "fee_schedules"
DROP INDEX `idx_fee_schedules_provider`;
-- reverse: create "fee_schedules" table
DROP TABLE `fee_schedules`;
-- reverse: create index "idx_procedure_codes_category" to table: "procedure_codes"
DROP INDEX `idx_procedure_codes_category`;
-- reverse: create "procedure_codes" table
DROP TABLE `procedure_codes`;
-- reverse: create index "idx_bundles_shortname" to table: "treatment_bundles"
DROP INDEX `idx_bundles_shortname`;
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
-- reverse: create index "idx_claims_appointment" to table: "claims"
DROP INDEX `idx_claims_appointment`;
-- reverse: create index "idx_claims_provider" to table: "claims"
DROP INDEX `idx_claims_provider`;
-- reverse: create index "idx_claims_patient" to table: "claims"
DROP INDEX `idx_claims_patient`;
-- reverse: create "claims" table
DROP TABLE `claims`;
