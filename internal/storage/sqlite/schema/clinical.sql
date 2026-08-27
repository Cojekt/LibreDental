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

CREATE TABLE IF NOT EXISTS dental_conditions (
	id TEXT PRIMARY KEY,
	patient_id TEXT NOT NULL,
	tooth_number INTEGER NOT NULL,
	surfaces TEXT DEFAULT '[]',
	ada_code TEXT DEFAULT '',
	description TEXT DEFAULT '',
	status TEXT NOT NULL,
	fee INTEGER DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY(patient_id) REFERENCES patients(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_dental_conditions_patient ON dental_conditions(patient_id);
CREATE INDEX IF NOT EXISTS idx_dental_conditions_tooth ON dental_conditions(patient_id, tooth_number);
