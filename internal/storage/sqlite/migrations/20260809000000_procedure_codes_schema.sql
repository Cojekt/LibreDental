-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS procedure_codes (
	country_code TEXT NOT NULL,
	code TEXT NOT NULL,
	category TEXT NOT NULL,
	description TEXT NOT NULL,
	default_fee REAL NOT NULL DEFAULT 0.0,
	is_active INTEGER NOT NULL DEFAULT 1,
	PRIMARY KEY (country_code, code)
);

CREATE INDEX IF NOT EXISTS idx_procedure_codes_country ON procedure_codes(country_code);
CREATE INDEX IF NOT EXISTS idx_procedure_codes_category ON procedure_codes(country_code, category);

CREATE TABLE IF NOT EXISTS fee_schedules (
	id TEXT PRIMARY KEY,
	country_code TEXT NOT NULL,
	code TEXT NOT NULL,
	provider_id TEXT DEFAULT '',
	custom_fee REAL NOT NULL,
	updated_at DATETIME NOT NULL,
	UNIQUE (country_code, code, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_fee_schedules_lookup ON fee_schedules(country_code, code, provider_id);

-- Seed US (CDT Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('US', 'D0120', 'Diagnostic', 'Periodic Oral Evaluation', 65.00),
('US', 'D0140', 'Diagnostic', 'Limited Oral Evaluation - Problem Focused', 85.00),
('US', 'D0210', 'Diagnostic', 'Intraoral - Comprehensive Series of Radiographic Images', 140.00),
('US', 'D0220', 'Diagnostic', 'Intraoral - Periapical First Radiographic Image', 35.00),
('US', 'D0274', 'Diagnostic', 'Bitewings - Four Radiographic Images', 70.00),
('US', 'D1110', 'Preventive', 'Prophylaxis - Adult', 110.00),
('US', 'D1120', 'Preventive', 'Prophylaxis - Child', 75.00),
('US', 'D1206', 'Preventive', 'Topical Application of Fluoride Varnish', 45.00),
('US', 'D1351', 'Preventive', 'Dental Sealant - Per Tooth', 55.00),
('US', 'D2140', 'Restorative', 'Amalgam - One Surface, Primary or Permanent', 125.00),
('US', 'D2150', 'Restorative', 'Amalgam - Two Surfaces, Primary or Permanent', 155.00),
('US', 'D2391', 'Restorative', 'Resin-Based Composite - One Surface, Posterior', 140.00),
('US', 'D2392', 'Restorative', 'Resin-Based Composite - Two Surfaces, Posterior', 185.00),
('US', 'D2393', 'Restorative', 'Resin-Based Composite - Three Surfaces, Posterior', 230.00),
('US', 'D2740', 'Restorative', 'Crown - Porcelain/Ceramic Substrate', 1100.00),
('US', 'D2750', 'Restorative', 'Crown - Porcelain Fused to High Noble Metal', 950.00),
('US', 'D3310', 'Endodontics', 'Endodontic Therapy - Anterior Tooth', 650.00),
('US', 'D3320', 'Endodontics', 'Endodontic Therapy - Bicuspid Tooth', 750.00),
('US', 'D3330', 'Endodontics', 'Endodontic Therapy - Molar Tooth', 950.00),
('US', 'D4210', 'Periodontics', 'Gingivectomy or Gingivoplasty - Four or More Teeth', 450.00),
('US', 'D4341', 'Periodontics', 'Periodontal Scaling and Root Planing - Four or More Teeth', 260.00),
('US', 'D5110', 'Prosthodontics', 'Complete Denture - Maxillary', 1400.00),
('US', 'D5120', 'Prosthodontics', 'Complete Denture - Mandibular', 1400.00),
('US', 'D7140', 'Oral Surgery', 'Extraction, Erupted Tooth or Exposed Root', 170.00),
('US', 'D7210', 'Oral Surgery', 'Extraction, Erupted Tooth Requiring Surgical Removal', 290.00);

-- Seed CA (Canadian Dental Association / Provincial Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('CA', '01201', 'Diagnostic', 'Examination - Recall', 75.00),
('CA', '01202', 'Diagnostic', 'Examination - Specific Emergency', 95.00),
('CA', '02102', 'Diagnostic', 'Radiographs - Complete Series', 160.00),
('CA', '02144', 'Diagnostic', 'Radiographs - Bitewing 4 films', 80.00),
('CA', '11101', 'Preventive', 'Polishing - 1 unit of time', 65.00),
('CA', '11111', 'Preventive', 'Scaling - 1 unit of time', 70.00),
('CA', '12101', 'Preventive', 'Fluoride Treatment - Topical Application', 40.00),
('CA', '13401', 'Preventive', 'Pit and Fissure Sealant - First Tooth', 50.00),
('CA', '21211', 'Restorative', 'Composite Restoration - 1 Surface Posterior', 160.00),
('CA', '21212', 'Restorative', 'Composite Restoration - 2 Surface Posterior', 210.00),
('CA', '21213', 'Restorative', 'Composite Restoration - 3 Surface Posterior', 260.00),
('CA', '27201', 'Restorative', 'Crown - Porcelain/Ceramic', 1150.00),
('CA', '27211', 'Restorative', 'Crown - Porcelain Fused to Metal', 1050.00),
('CA', '33111', 'Endodontics', 'Root Canal Therapy - 1 Canal', 700.00),
('CA', '33121', 'Endodontics', 'Root Canal Therapy - 2 Canals', 820.00),
('CA', '33131', 'Endodontics', 'Root Canal Therapy - 3 Canals', 1050.00),
('CA', '43421', 'Periodontics', 'Scaling and Root Planing - 1 Unit', 95.00),
('CA', '51101', 'Prosthodontics', 'Complete Upper Denture', 1500.00),
('CA', '71101', 'Oral Surgery', 'Extraction - Uncomplicated Erupted Tooth', 180.00),
('CA', '72101', 'Oral Surgery', 'Extraction - Surgical Removal', 320.00);

-- Seed GB (NHS / BDA Billing Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('GB', 'BAND1', 'Diagnostic', 'NHS Band 1 - Examination, Diagnosis & Advice', 26.80),
('GB', 'BAND2', 'Restorative', 'NHS Band 2 - Fillings, Root Canal, Extractions', 73.50),
('GB', 'BAND3', 'Prosthodontics', 'NHS Band 3 - Crowns, Bridges, Dentures', 319.10),
('GB', 'BAND4', 'Diagnostic', 'NHS Urgent Care Band', 26.80),
('GB', 'EXAM01', 'Diagnostic', 'Private Dental Examination & X-Rays', 55.00),
('GB', 'HYG01', 'Preventive', 'Hygienist Appointment & Scaling', 75.00),
('GB', 'COMP01', 'Restorative', 'Private White Composite Filling - 1 Surface', 120.00),
('GB', 'COMP02', 'Restorative', 'Private White Composite Filling - 2 Surface', 160.00),
('GB', 'CRN01', 'Restorative', 'Private Porcelain Crown', 650.00),
('GB', 'RCT01', 'Endodontics', 'Private Root Canal Treatment - Molar', 550.00),
('GB', 'EXT01', 'Oral Surgery', 'Private Simple Extraction', 130.00);

-- Seed AU (Australian Dental Association Item Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('AU', '011', 'Diagnostic', 'Comprehensive Oral Examination', 80.00),
('AU', '012', 'Diagnostic', 'Periodic Oral Examination', 65.00),
('AU', '022', 'Diagnostic', 'Intraoral Periapical X-ray', 45.00),
('AU', '111', 'Preventive', 'Dental Prophylaxis / Cleaning', 120.00),
('AU', '121', 'Preventive', 'Topical Fluoride Application', 40.00),
('AU', '161', 'Preventive', 'Fissure Sealing - Per Tooth', 60.00),
('AU', '531', 'Restorative', 'Adhesive Restoration - 1 Surface Rear Tooth', 165.00),
('AU', '532', 'Restorative', 'Adhesive Restoration - 2 Surface Rear Tooth', 215.00),
('AU', '533', 'Restorative', 'Adhesive Restoration - 3 Surface Rear Tooth', 265.00),
('AU', '615', 'Restorative', 'Crown - Porcelain Ceramic', 1450.00),
('AU', '415', 'Endodontics', 'Complete Root Canal Treatment - 1 Canal', 750.00),
('AU', '417', 'Endodontics', 'Complete Root Canal Treatment - 3 Canals', 1150.00),
('AU', '311', 'Oral Surgery', 'Extraction of Tooth', 210.00);

-- Seed DE (GOZ / BEMA German Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('DE', 'GOZ 0010', 'Diagnostic', 'Eingehende Untersuchung zur Feststellung von Zahn- Krankheiten', 35.00),
('DE', 'GOZ 0070', 'Diagnostic', 'Vitalitätsprüfung eines Zahnes', 15.00),
('DE', 'GOZ 1040', 'Preventive', 'Professionelle Zahnreinigung (PZR) pro Zahn', 95.00),
('DE', 'GOZ 2060', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 1-flächig', 110.00),
('DE', 'GOZ 2080', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 2-flächig', 145.00),
('DE', 'GOZ 2100', 'Restorative', 'Präparieren einer Kavität und Restauration mit Komposit - 3-flächig', 180.00),
('DE', 'GOZ 2210', 'Restorative', 'Vollkeramische Krone', 750.00),
('DE', 'GOZ 2410', 'Endodontics', 'Wurzelkanalaufbereitung pro Kanal', 480.00),
('DE', 'GOZ 3020', 'Oral Surgery', 'Entfernung eines einwurzeligen Zahnes', 85.00),
('DE', 'GOZ 3030', 'Oral Surgery', 'Entfernung eines mehrwurzeligen Zahnes', 125.00);

-- Seed FR (CCAM French Codes)
INSERT OR IGNORE INTO procedure_codes (country_code, code, category, description, default_fee) VALUES
('FR', 'HBQD001', 'Diagnostic', 'Examen bucco-dentaire', 30.00),
('FR', 'HBDD002', 'Preventive', 'Détartrage et nettoyage dentaire complet', 43.38),
('FR', 'HBMD038', 'Restorative', 'Restauration d un dent sur 1 face par matériau composite', 65.00),
('FR', 'HBMD044', 'Restorative', 'Restauration d un dent sur 2 faces par matériau composite', 85.00),
('FR', 'HBMD050', 'Restorative', 'Restauration d un dent sur 3 faces ou plus par matériau composite', 115.00),
('FR', 'HBLD038', 'Restorative', 'Pose d une couronne dentaire céramo-métallique', 500.00),
('FR', 'HBFD033', 'Endodontics', 'Exérèse du contenu canalaire d une molaire (Dévitalisation)', 250.00),
('FR', 'HBGD036', 'Oral Surgery', 'Extraction d une dent permanente sur arcade', 80.00);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fee_schedules;
DROP TABLE IF EXISTS procedure_codes;
-- +goose StatementEnd
