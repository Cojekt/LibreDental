-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS claims (
    id TEXT PRIMARY KEY,
    patient_id TEXT NOT NULL,
    provider_id TEXT NOT NULL DEFAULT '',
    appointment_id TEXT DEFAULT '',
    insurance_carrier TEXT DEFAULT '',
    policy_number TEXT DEFAULT '',
    group_number TEXT DEFAULT '',
    date_of_service TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    notes TEXT DEFAULT '',
    line_items TEXT DEFAULT '[]',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_claims_patient ON claims(patient_id);
CREATE INDEX IF NOT EXISTS idx_claims_status ON claims(status);
CREATE INDEX IF NOT EXISTS idx_claims_date ON claims(date_of_service);

CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    patient_id TEXT NOT NULL,
    claim_id TEXT DEFAULT '',
    amount INTEGER NOT NULL,
    method TEXT NOT NULL DEFAULT 'cash',
    date TEXT NOT NULL,
    notes TEXT DEFAULT '',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_payments_patient ON payments(patient_id);
CREATE INDEX IF NOT EXISTS idx_payments_claim ON payments(claim_id);
CREATE INDEX IF NOT EXISTS idx_payments_date ON payments(date);

CREATE TABLE IF NOT EXISTS treatment_bundles (
    id TEXT PRIMARY KEY,
    shortname TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    items TEXT DEFAULT '[]',
    total_fee INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_bundles_shortname ON treatment_bundles(shortname);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS treatment_bundles;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS claims;
-- +goose StatementEnd
