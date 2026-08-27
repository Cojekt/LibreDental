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

INSERT OR IGNORE INTO country_configs (code, name, national_id_name, national_id_type, national_id_placeholder, default_tooth_system, default_currency, state_province_label, postal_code_label, date_format, is_default) VALUES
('US', 'United States', 'Social Security Number (SSN)', 'ssn', '000-00-0000', 'universal', 'USD', 'State', 'ZIP Code', 'MM/DD/YYYY', 1),
('CA', 'Canada', 'Social Insurance Number (SIN)', 'sin', '000-000-000', 'fdi', 'CAD', 'Province', 'Postal Code', 'YYYY-MM-DD', 0),
('GB', 'United Kingdom', 'NHS Number', 'nhs_number', '000 000 0000', 'fdi', 'GBP', 'County', 'Postcode', 'DD/MM/YYYY', 0),
('AU', 'Australia', 'Medicare Card Number', 'medicare_num', '0000 00000 0', 'fdi', 'AUD', 'State', 'Postcode', 'DD/MM/YYYY', 0),
('DE', 'Germany', 'Tax / Health Insurance ID', 'tax_id', 'X000000000', 'fdi', 'EUR', 'State / Bundesland', 'PLZ (Postal Code)', 'DD.MM.YYYY', 0),
('FR', 'France', 'NIR (NIR / Numéro SS)', 'nir', '1 00 00 00 000 000 00', 'fdi', 'EUR', 'Region / Department', 'Code Postal', 'DD/MM/YYYY', 0);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('US', 'D0120', 'Diagnostic', 'Periodic Oral Evaluation', 6500),
('US', 'D0140', 'Diagnostic', 'Limited Oral Evaluation - Problem Focused', 8500),
('US', 'D0210', 'Diagnostic', 'Intraoral - Comprehensive Series of Radiographic Images', 14000),
('US', 'D0220', 'Diagnostic', 'Intraoral - Periapical First Radiographic Image', 3500),
('US', 'D0274', 'Diagnostic', 'Bitewings - Four Radiographic Images', 7000),
('US', 'D1110', 'Preventive', 'Prophylaxis - Adult', 11000),
('US', 'D1120', 'Preventive', 'Prophylaxis - Child', 7500),
('US', 'D1206', 'Preventive', 'Topical Application of Fluoride Varnish', 4500),
('US', 'D1351', 'Preventive', 'Dental Sealant - Per Tooth', 5500),
('US', 'D2140', 'Restorative', 'Amalgam - One Surface, Primary or Permanent', 12500),
('US', 'D2150', 'Restorative', 'Amalgam - Two Surfaces, Primary or Permanent', 15500),
('US', 'D2391', 'Restorative', 'Resin-Based Composite - One Surface, Posterior', 14000),
('US', 'D2392', 'Restorative', 'Resin-Based Composite - Two Surfaces, Posterior', 18500),
('US', 'D2393', 'Restorative', 'Resin-Based Composite - Three Surfaces, Posterior', 23000),
('US', 'D2740', 'Restorative', 'Crown - Porcelain/Ceramic Substrate', 110000),
('US', 'D2750', 'Restorative', 'Crown - Porcelain Fused to High Noble Metal', 95000),
('US', 'D3310', 'Endodontics', 'Endodontic Therapy - Anterior Tooth', 65000),
('US', 'D3320', 'Endodontics', 'Endodontic Therapy - Bicuspid Tooth', 75000),
('US', 'D3330', 'Endodontics', 'Endodontic Therapy - Molar Tooth', 95000),
('US', 'D4210', 'Periodontics', 'Gingivectomy or Gingivoplasty - Four or More Teeth', 45000),
('US', 'D4341', 'Periodontics', 'Periodontal Scaling and Root Planing - Four or More Teeth', 26000),
('US', 'D5110', 'Prosthodontics', 'Complete Denture - Maxillary', 140000),
('US', 'D5120', 'Prosthodontics', 'Complete Denture - Mandibular', 140000),
('US', 'D7140', 'Oral Surgery', 'Extraction, Erupted Tooth or Exposed Root', 17000),
('US', 'D7210', 'Oral Surgery', 'Extraction, Erupted Tooth Requiring Surgical Removal', 29000);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('CA', '01201', 'Diagnostic', 'Examination - Recall', 7500),
('CA', '01202', 'Diagnostic', 'Examination - Specific Emergency', 9500),
('CA', '02102', 'Diagnostic', 'Radiographs - Complete Series', 16000),
('CA', '02144', 'Diagnostic', 'Radiographs - Bitewing 4 films', 8000),
('CA', '11101', 'Preventive', 'Polishing - 1 unit of time', 6500),
('CA', '11111', 'Preventive', 'Scaling - 1 unit of time', 7000),
('CA', '12101', 'Preventive', 'Fluoride Treatment - Topical Application', 4000),
('CA', '13401', 'Preventive', 'Pit and Fissure Sealant - First Tooth', 5000),
('CA', '21211', 'Restorative', 'Composite Restoration - 1 Surface Posterior', 16000),
('CA', '21212', 'Restorative', 'Composite Restoration - 2 Surface Posterior', 21000),
('CA', '21213', 'Restorative', 'Composite Restoration - 3 Surface Posterior', 26000),
('CA', '27201', 'Restorative', 'Crown - Porcelain/Ceramic', 115000),
('CA', '27211', 'Restorative', 'Crown - Porcelain Fused to Metal', 105000),
('CA', '33111', 'Endodontics', 'Root Canal Therapy - 1 Canal', 70000),
('CA', '33121', 'Endodontics', 'Root Canal Therapy - 2 Canals', 82000),
('CA', '33131', 'Endodontics', 'Root Canal Therapy - 3 Canals', 105000),
('CA', '43421', 'Periodontics', 'Scaling and Root Planing - 1 Unit', 9500),
('CA', '51101', 'Prosthodontics', 'Complete Upper Denture', 150000),
('CA', '71101', 'Oral Surgery', 'Extraction - Uncomplicated Erupted Tooth', 18000),
('CA', '72101', 'Oral Surgery', 'Extraction - Surgical Removal', 32000);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('GB', 'BAND1', 'Diagnostic', 'NHS Band 1 - Examination, Diagnosis & Advice', 2680),
('GB', 'BAND2', 'Restorative', 'NHS Band 2 - Fillings, Root Canal, Extractions', 7350),
('GB', 'BAND3', 'Prosthodontics', 'NHS Band 3 - Crowns, Bridges, Dentures', 31910),
('GB', 'BAND4', 'Diagnostic', 'NHS Urgent Care Band', 2680),
('GB', 'EXAM01', 'Diagnostic', 'Private Dental Examination & X-Rays', 5500),
('GB', 'HYG01', 'Preventive', 'Hygienist Appointment & Scaling', 7500),
('GB', 'COMP01', 'Restorative', 'Private White Composite Filling - 1 Surface', 12000),
('GB', 'COMP02', 'Restorative', 'Private White Composite Filling - 2 Surface', 16000),
('GB', 'CRN01', 'Restorative', 'Private Porcelain Crown', 65000),
('GB', 'RCT01', 'Endodontics', 'Private Root Canal Treatment - Molar', 55000),
('GB', 'EXT01', 'Oral Surgery', 'Private Simple Extraction', 13000);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('AU', '011', 'Diagnostic', 'Comprehensive Oral Examination', 8000),
('AU', '012', 'Diagnostic', 'Periodic Oral Examination', 6500),
('AU', '022', 'Diagnostic', 'Intraoral Periapical X-ray', 4500),
('AU', '111', 'Preventive', 'Dental Prophylaxis / Cleaning', 12000),
('AU', '121', 'Preventive', 'Topical Fluoride Application', 4000),
('AU', '161', 'Preventive', 'Fissure Sealing - Per Tooth', 6000),
('AU', '531', 'Restorative', 'Adhesive Restoration - 1 Surface Rear Tooth', 16500),
('AU', '532', 'Restorative', 'Adhesive Restoration - 2 Surface Rear Tooth', 21500),
('AU', '533', 'Restorative', 'Adhesive Restoration - 3 Surface Rear Tooth', 26500),
('AU', '615', 'Restorative', 'Crown - Porcelain Ceramic', 145000),
('AU', '415', 'Endodontics', 'Complete Root Canal Treatment - 1 Canal', 75000),
('AU', '417', 'Endodontics', 'Complete Root Canal Treatment - 3 Canals', 115000),
('AU', '311', 'Oral Surgery', 'Extraction of Tooth', 21000);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('DE', 'GOZ 0010', 'Diagnostic', 'Eingehende Untersuchung zur Feststellung von Zahn- Krankheiten', 3500),
('DE', 'GOZ 0070', 'Diagnostic', 'Vitalitätsprüfung eines Zahnes', 1500),
('DE', 'GOZ 1040', 'Preventive', 'Professionelle Zahnreinigung (PZR) pro Zahn', 9500),
('DE', 'GOZ 2060', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 1-flächig', 11000),
('DE', 'GOZ 2080', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 2-flächig', 14500),
('DE', 'GOZ 2100', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 3-flächig', 18000),
('DE', 'GOZ 2210', 'Restorative', 'Vollkeramische Krone', 75000),
('DE', 'GOZ 2410', 'Endodontics', 'Wurzelkanalaufbereitung pro Kanal', 48000),
('DE', 'GOZ 3020', 'Oral Surgery', 'Entfernung eines einwurzeligen Zahnes', 8500),
('DE', 'GOZ 3030', 'Oral Surgery', 'Entfernung eines mehrwurzeligen Zahnes', 12500);

INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('FR', 'HBQD001', 'Diagnostic', 'Examen bucco-dentaire', 3000),
('FR', 'HBDD002', 'Preventive', 'Détartrage et nettoyage dentaire complet', 4338),
('FR', 'HBMD038', 'Restorative', 'Restauration d un dent sur 1 face par matériau composite', 6500),
('FR', 'HBMD044', 'Restorative', 'Restauration d un dent sur 2 faces par matériau composite', 8500),
('FR', 'HBMD050', 'Restorative', 'Restauration d un dent sur 3 faces ou plus par matériau composite', 11500),
('FR', 'HBLD038', 'Restorative', 'Pose d une couronne dentaire céramo-métallique', 50000),
('FR', 'HBFD033', 'Endodontics', 'Exérèse du contenu canalaire d une molaire (Dévitalisation)', 25000),
('FR', 'HBGD036', 'Oral Surgery', 'Extraction d une dent permanente sur arcade', 8000);

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
