package seed

import (
	"database/sql"
	"fmt"
)

const seedData = `
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
`

// Run executes the default seed data for the application.
func Run(db *sql.DB) error {
	_, err := db.Exec(seedData)
	if err != nil {
		return fmt.Errorf("failed to insert seed data: %w", err)
	}
	return nil
}
