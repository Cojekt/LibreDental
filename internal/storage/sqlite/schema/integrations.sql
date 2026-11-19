CREATE TABLE IF NOT EXISTS program_bridges (
	name TEXT NOT NULL PRIMARY KEY,
	enabled INTEGER NOT NULL DEFAULT 0,
	path TEXT NOT NULL DEFAULT '',
	args TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL CHECK (typeof(created_at) = 'integer'),
	updated_at INTEGER NOT NULL CHECK (typeof(updated_at) = 'integer')
);
