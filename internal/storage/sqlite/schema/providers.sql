CREATE TABLE IF NOT EXISTS providers (
	id TEXT NOT NULL PRIMARY KEY,
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
	created_at INTEGER NOT NULL CHECK (typeof(created_at) = 'integer'),
	updated_at INTEGER NOT NULL CHECK (typeof(updated_at) = 'integer'),
	npi TEXT DEFAULT '',
	taxonomy_code TEXT DEFAULT ''
);

CREATE TABLE IF NOT EXISTS timecards (
    id TEXT NOT NULL PRIMARY KEY,
    provider_id TEXT NOT NULL,
    clock_in INTEGER NOT NULL CHECK (typeof(clock_in) = 'integer'),
    clock_out INTEGER NULL CHECK (clock_out IS NULL OR typeof(clock_out) = 'integer'),
    hourly_rate INTEGER NOT NULL DEFAULT 0,
    total_minutes INTEGER,
    total_pay INTEGER GENERATED ALWAYS AS ((total_minutes * hourly_rate + 30) / 60) VIRTUAL,
    paid_at INTEGER NULL CHECK (paid_at IS NULL OR typeof(paid_at) = 'integer'),
    is_manual BOOLEAN NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL CHECK (typeof(created_at) = 'integer'),
    updated_at INTEGER NOT NULL CHECK (typeof(updated_at) = 'integer'),
    FOREIGN KEY (provider_id) REFERENCES providers (id)
);

CREATE INDEX IF NOT EXISTS idx_timecards_provider_clockin ON timecards (provider_id, clock_in);
CREATE UNIQUE INDEX IF NOT EXISTS idx_timecards_active_provider ON timecards (provider_id) WHERE clock_out IS NULL;
