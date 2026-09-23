CREATE TABLE IF NOT EXISTS notification_log (
    id TEXT NOT NULL PRIMARY KEY,
    patient_id TEXT NOT NULL,
    appointment_id TEXT,
    channel TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    recipient TEXT NOT NULL,
    subject TEXT DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    error_message TEXT DEFAULT '',
    sent_at DATETIME NOT NULL,
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE RESTRICT,
    FOREIGN KEY (appointment_id) REFERENCES appointments(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_log_patient ON notification_log(patient_id);
CREATE INDEX IF NOT EXISTS idx_notification_log_appointment ON notification_log(appointment_id);
CREATE INDEX IF NOT EXISTS idx_notification_log_sent_at ON notification_log(sent_at);
