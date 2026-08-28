CREATE TABLE IF NOT EXISTS patients (
	id TEXT NOT NULL PRIMARY KEY,
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

CREATE TABLE IF NOT EXISTS documents (
    id TEXT NOT NULL PRIMARY KEY,
    patient_id TEXT,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    file_path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    content_type TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_documents_patient_id ON documents (patient_id);
