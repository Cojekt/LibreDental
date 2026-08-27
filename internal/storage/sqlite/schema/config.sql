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

CREATE TABLE IF NOT EXISTS operatories (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	room_code TEXT DEFAULT '',
	type TEXT NOT NULL DEFAULT 'general',
	is_active INTEGER NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

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
