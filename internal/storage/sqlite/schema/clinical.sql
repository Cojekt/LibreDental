CREATE TABLE IF NOT EXISTS appointments (
	id TEXT NOT NULL PRIMARY KEY,
	patient_id TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	operatory_id TEXT NOT NULL,
	start_time INTEGER NOT NULL CHECK (typeof(start_time) = 'integer'),
	end_time INTEGER NOT NULL CHECK (typeof(end_time) = 'integer'),
	status TEXT NOT NULL,
	reason TEXT DEFAULT '',
	color TEXT DEFAULT '',
	notes TEXT DEFAULT '',
	created_at INTEGER NOT NULL CHECK (typeof(created_at) = 'integer'),
	updated_at INTEGER NOT NULL CHECK (typeof(updated_at) = 'integer'),
	version INTEGER NOT NULL DEFAULT 1,
	FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE RESTRICT,
	FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE RESTRICT,
	FOREIGN KEY (operatory_id) REFERENCES operatories(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_appointments_patient ON appointments(patient_id);
CREATE INDEX IF NOT EXISTS idx_appointments_provider ON appointments(provider_id, start_time);
CREATE INDEX IF NOT EXISTS idx_appointments_operatory ON appointments(operatory_id, start_time);
CREATE INDEX IF NOT EXISTS idx_appointments_date ON appointments(start_time, end_time);

CREATE TABLE IF NOT EXISTS dental_conditions (
	id TEXT NOT NULL PRIMARY KEY,
	patient_id TEXT NOT NULL,
	tooth_number INTEGER NOT NULL,
	surfaces TEXT DEFAULT '[]',
	ada_code TEXT DEFAULT '',
	description TEXT DEFAULT '',
	status TEXT NOT NULL,
	fee INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL CHECK (typeof(created_at) = 'integer'),
	updated_at INTEGER NOT NULL CHECK (typeof(updated_at) = 'integer'),
 insurance_provider_id TEXT REFERENCES insurance_providers(id) ON DELETE RESTRICT,
 pricing_source TEXT CHECK(pricing_source IS NULL OR json_valid(pricing_source)),
	FOREIGN KEY(patient_id) REFERENCES patients(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_dental_conditions_patient ON dental_conditions(patient_id);
CREATE INDEX IF NOT EXISTS idx_dental_conditions_tooth ON dental_conditions(patient_id, tooth_number);
