CREATE TABLE IF NOT EXISTS providers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'dentist',
	specialty TEXT DEFAULT '',
	license_number TEXT DEFAULT '',
	email TEXT DEFAULT '',
	phone TEXT DEFAULT '',
	color TEXT DEFAULT '#3b82f6',
	pin TEXT DEFAULT '',
	is_active INTEGER NOT NULL DEFAULT 1,
	hourly_rate INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS timecards (
    id TEXT PRIMARY KEY,
    provider_id TEXT NOT NULL,
    clock_in DATETIME NOT NULL,
    clock_out DATETIME,
    hourly_rate INTEGER NOT NULL DEFAULT 0,
    total_minutes INTEGER,
    total_pay INTEGER,
    paid_at DATETIME,
    is_manual BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (provider_id) REFERENCES providers (id)
);

CREATE INDEX IF NOT EXISTS idx_timecards_provider_clockin ON timecards (provider_id, clock_in);
CREATE UNIQUE INDEX IF NOT EXISTS idx_timecards_active_provider ON timecards (provider_id) WHERE clock_out IS NULL;
