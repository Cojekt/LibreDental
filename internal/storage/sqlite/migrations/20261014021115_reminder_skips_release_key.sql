-- +goose Up
-- Hand-edited from `atlas migrate diff` output: Atlas rebuilt `notification_log` (copy, DROP
-- TABLE, rename) although only the reminder index's WHERE clause changed. Recreating the index
-- in place reaches the same schema without moving any rows; `atlas migrate diff` reports no
-- drift.
-- Skipped reminders no longer hold the reminder key, so a reminder skipped for a fixable reason
-- (such as an invalid phone number) can still be sent once the data is corrected.
DROP INDEX `idx_notification_log_reminder`;
CREATE UNIQUE INDEX `idx_notification_log_reminder` ON `notification_log` (`appointment_id`, `appointment_start`, `reminder_kind`, `channel`) WHERE reminder_kind IS NOT NULL AND status <> 'skipped';

-- +goose Down
-- Restores the stricter index. It fails, without changing any data, if a reminder was sent
-- after an earlier attempt for it was skipped; that history can't be represented under the old
-- index.
DROP INDEX `idx_notification_log_reminder`;
CREATE UNIQUE INDEX `idx_notification_log_reminder` ON `notification_log` (`appointment_id`, `appointment_start`, `reminder_kind`, `channel`) WHERE reminder_kind IS NOT NULL;
