-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS practice_config (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	clinic_name TEXT NOT NULL DEFAULT 'My Dental Clinic',
	tagline TEXT DEFAULT '',
	tax_id TEXT DEFAULT '',
	license_number TEXT DEFAULT '',
	phone TEXT DEFAULT '',
	email TEXT DEFAULT '',
	website TEXT DEFAULT '',
	address_line1 TEXT DEFAULT '',
	address_line2 TEXT DEFAULT '',
	city TEXT DEFAULT '',
	state_province TEXT DEFAULT '',
	postal_code TEXT DEFAULT '',
	country_code TEXT NOT NULL,
	currency TEXT NOT NULL,
	tooth_system TEXT NOT NULL DEFAULT 'universal',
	date_format TEXT NOT NULL DEFAULT 'YYYY-MM-DD',
	business_hours TEXT DEFAULT '[]',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS providers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'dentist',
	specialty TEXT DEFAULT '',
	license_number TEXT DEFAULT '',
	email TEXT DEFAULT '',
	phone TEXT DEFAULT '',
	color TEXT DEFAULT '#3b82f6',
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS operatories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	room_code TEXT DEFAULT '',
	type TEXT NOT NULL DEFAULT 'general',
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);


CREATE TABLE IF NOT EXISTS patients (
	id TEXT PRIMARY KEY,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	middle_name TEXT DEFAULT '',
	preferred_name TEXT DEFAULT '',
	date_of_birth TEXT NOT NULL,
	sex TEXT NOT NULL,
	email TEXT DEFAULT '',
	phone_primary TEXT DEFAULT '',
	phone_secondary TEXT DEFAULT '',
	emergency_contact_name TEXT DEFAULT '',
	emergency_contact_rel TEXT DEFAULT '',
	emergency_contact_phone TEXT DEFAULT '',
	guarantor_name TEXT DEFAULT '',
	guarantor_rel TEXT DEFAULT '',
	guarantor_phone TEXT DEFAULT '',
	insurance_carrier TEXT DEFAULT '',
	insurance_policy_number TEXT DEFAULT '',
	insurance_group_number TEXT DEFAULT '',
	preferred_contact_method TEXT DEFAULT 'phone',
	preferred_language TEXT DEFAULT '',
	reminder_opt_in INTEGER NOT NULL DEFAULT 1,
	preferred_provider_id TEXT DEFAULT '',
	referral_source TEXT DEFAULT '',
	address_line1 TEXT DEFAULT '',
	address_line2 TEXT DEFAULT '',
	city TEXT DEFAULT '',
	state_province TEXT DEFAULT '',
	postal_code TEXT DEFAULT '',
	country_code TEXT DEFAULT '',
	national_id_type TEXT DEFAULT '',
	national_id TEXT DEFAULT '',
	medical_alerts TEXT DEFAULT '[]',
	allergies TEXT DEFAULT '[]',
	notes TEXT DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	status TEXT NOT NULL DEFAULT 'active'
);

CREATE INDEX IF NOT EXISTS idx_patients_name ON patients(last_name, first_name);
CREATE INDEX IF NOT EXISTS idx_patients_dob ON patients(date_of_birth);

CREATE TABLE IF NOT EXISTS appointments (
	id TEXT PRIMARY KEY,
	patient_id TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	operatory_id TEXT NOT NULL,
	start_time DATETIME NOT NULL,
	end_time DATETIME NOT NULL,
	status TEXT NOT NULL,
	reason TEXT DEFAULT '',
	color TEXT DEFAULT '',
	notes TEXT DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	version INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_appointments_patient ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_appointments_date ON appointments(start_time, end_time);

CREATE TABLE IF NOT EXISTS audit_logs (
	id TEXT PRIMARY KEY,
	timestamp DATETIME NOT NULL,
	user_id TEXT NOT NULL,
	user_name TEXT NOT NULL,
	patient_id TEXT DEFAULT '',
	action TEXT NOT NULL,
	resource TEXT NOT NULL,
	resource_id TEXT DEFAULT '',
	details TEXT DEFAULT '',
	ip_address TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dental_conditions (
	id TEXT PRIMARY KEY,
	patient_id TEXT NOT NULL,
	tooth_number INTEGER NOT NULL,
	surfaces TEXT DEFAULT '[]',
	ada_code TEXT DEFAULT '',
	description TEXT DEFAULT '',
	status TEXT NOT NULL,
	fee REAL DEFAULT 0.0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_audit_patient ON audit_logs(patient_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_dental_conditions_patient ON dental_conditions(patient_id);
CREATE INDEX IF NOT EXISTS idx_dental_conditions_tooth ON dental_conditions(patient_id, tooth_number);

CREATE TABLE IF NOT EXISTS country_configs (
	code TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	national_id_name TEXT NOT NULL,
	national_id_type TEXT NOT NULL,
	national_id_placeholder TEXT NOT NULL,
	default_tooth_system TEXT NOT NULL,
	default_currency TEXT NOT NULL,
	state_province_label TEXT NOT NULL,
	postal_code_label TEXT NOT NULL,
	date_format TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0
);

INSERT OR IGNORE INTO country_configs (code, name, national_id_name, national_id_type, national_id_placeholder, default_tooth_system, default_currency, state_province_label, postal_code_label, date_format, is_default) VALUES
('US', 'United States', 'Social Security Number (SSN)', 'ssn', '000-00-0000', 'universal', 'USD', 'State', 'ZIP Code', 'MM/DD/YYYY', 1),
('CA', 'Canada', 'Social Insurance Number (SIN)', 'sin', '000-000-000', 'fdi', 'CAD', 'Province', 'Postal Code', 'YYYY-MM-DD', 0),
('GB', 'United Kingdom', 'NHS Number', 'nhs_number', '000 000 0000', 'fdi', 'GBP', 'County', 'Postcode', 'DD/MM/YYYY', 0),
('AU', 'Australia', 'Medicare Card Number', 'medicare_num', '0000 00000 0', 'fdi', 'AUD', 'State', 'Postcode', 'DD/MM/YYYY', 0),
('DE', 'Germany', 'Tax / Health Insurance ID', 'tax_id', 'X000000000', 'fdi', 'EUR', 'State / Bundesland', 'PLZ (Postal Code)', 'DD.MM.YYYY', 0),
('FR', 'France', 'NIR (NIR / Numéro SS)', 'nir', '1 00 00 00 000 000 00', 'fdi', 'EUR', 'Region / Department', 'Code Postal', 'DD/MM/YYYY', 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS country_configs;
DROP TABLE IF EXISTS dental_conditions;
DROP TABLE IF EXISTS operatories;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS practice_config;
-- +goose StatementEnd

