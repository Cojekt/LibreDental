CREATE TABLE IF NOT EXISTS notification_log (
    id TEXT NOT NULL PRIMARY KEY,
    patient_id TEXT NOT NULL,
    appointment_id TEXT,
    channel TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    external_message_id TEXT DEFAULT '',
    recipient TEXT NOT NULL,
    subject TEXT DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    error_message TEXT DEFAULT '',
    sent_at DATETIME NOT NULL,
    reminder_kind TEXT NULL,
    appointment_start TEXT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE RESTRICT,
    FOREIGN KEY (appointment_id) REFERENCES appointments(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_log_patient ON notification_log(patient_id);
CREATE INDEX IF NOT EXISTS idx_notification_log_appointment ON notification_log(appointment_id);
CREATE INDEX IF NOT EXISTS idx_notification_log_sent_at ON notification_log(sent_at);

-- One automatic reminder per appointment time, rule timing, and channel, so a reminder is never
-- sent twice even with two processes running, while a rescheduled appointment can be reminded
-- again for its new time. Manual sends have no reminder_kind and are not constrained.
CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_log_reminder
    ON notification_log(appointment_id, appointment_start, reminder_kind, channel)
    WHERE reminder_kind IS NOT NULL;

CREATE TABLE IF NOT EXISTS reminder_settings (
    id INTEGER NOT NULL PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,
    sending_hours_start TEXT NOT NULL DEFAULT '08:00',
    sending_hours_end TEXT NOT NULL DEFAULT '20:00',
    enabled_at TEXT NULL,
    enabled_by TEXT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reminder_rules (
    id TEXT NOT NULL PRIMARY KEY,
    offset_minutes INTEGER NOT NULL,
    channel TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    subject_template TEXT NOT NULL DEFAULT '',
    body_template TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (offset_minutes, channel)
);
