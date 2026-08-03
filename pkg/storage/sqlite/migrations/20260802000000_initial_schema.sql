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
	gender TEXT NOT NULL,
	email TEXT DEFAULT '',
	phone_primary TEXT DEFAULT '',
	phone_secondary TEXT DEFAULT '',
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

CREATE INDEX IF NOT EXISTS idx_audit_patient ON audit_logs(patient_id);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS operatories;
DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS patients;
DROP TABLE IF EXISTS practice_config;
-- +goose StatementEnd
