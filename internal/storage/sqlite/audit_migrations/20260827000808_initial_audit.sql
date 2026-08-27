-- +goose Up
-- create "audit_logs" table
CREATE TABLE `audit_logs` (
  `id` text NULL,
  `timestamp` datetime NOT NULL,
  `user_id` text NOT NULL,
  `user_name` text NOT NULL,
  `patient_id` text NULL DEFAULT '',
  `action` text NOT NULL,
  `resource` text NOT NULL,
  `resource_id` text NULL DEFAULT '',
  `details` text NULL DEFAULT '',
  `ip_address` text NULL DEFAULT '',
  PRIMARY KEY (`id`)
);
-- create index "idx_audit_patient" to table: "audit_logs"
CREATE INDEX `idx_audit_patient` ON `audit_logs` (`patient_id`);
-- create index "idx_audit_timestamp" to table: "audit_logs"
CREATE INDEX `idx_audit_timestamp` ON `audit_logs` (`timestamp`);

-- +goose Down
-- reverse: create index "idx_audit_timestamp" to table: "audit_logs"
DROP INDEX `idx_audit_timestamp`;
-- reverse: create index "idx_audit_patient" to table: "audit_logs"
DROP INDEX `idx_audit_patient`;
-- reverse: create "audit_logs" table
DROP TABLE `audit_logs`;
