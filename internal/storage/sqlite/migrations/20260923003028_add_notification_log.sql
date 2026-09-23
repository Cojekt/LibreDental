-- +goose Up
-- create "notification_log" table
CREATE TABLE `notification_log` (
  `id` text NOT NULL,
  `patient_id` text NOT NULL,
  `appointment_id` text NULL,
  `channel` text NOT NULL,
  `provider_name` text NOT NULL,
  `recipient` text NOT NULL,
  `subject` text NULL DEFAULT '',
  `body` text NOT NULL DEFAULT '',
  `status` text NOT NULL,
  `error_message` text NULL DEFAULT '',
  `sent_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CONSTRAINT `0` FOREIGN KEY (`appointment_id`) REFERENCES `appointments` (`id`) ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT `1` FOREIGN KEY (`patient_id`) REFERENCES `patients` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT
);
-- create index "idx_notification_log_patient" to table: "notification_log"
CREATE INDEX `idx_notification_log_patient` ON `notification_log` (`patient_id`);
-- create index "idx_notification_log_appointment" to table: "notification_log"
CREATE INDEX `idx_notification_log_appointment` ON `notification_log` (`appointment_id`);
-- create index "idx_notification_log_sent_at" to table: "notification_log"
CREATE INDEX `idx_notification_log_sent_at` ON `notification_log` (`sent_at`);

-- +goose Down
-- reverse: create index "idx_notification_log_sent_at" to table: "notification_log"
DROP INDEX `idx_notification_log_sent_at`;
-- reverse: create index "idx_notification_log_appointment" to table: "notification_log"
DROP INDEX `idx_notification_log_appointment`;
-- reverse: create index "idx_notification_log_patient" to table: "notification_log"
DROP INDEX `idx_notification_log_patient`;
-- reverse: create "notification_log" table
DROP TABLE `notification_log`;
