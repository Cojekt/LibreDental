-- +goose Up
-- Hand-edited from `atlas migrate diff` output: Atlas rebuilt `patients` (copy, DROP TABLE,
-- rename) to drop insurance_subscriber_id, which fails on any install with claims or charts
-- because goose runs migrations in a transaction, where PRAGMA foreign_keys = off is a no-op.
-- ALTER TABLE DROP COLUMN reaches the same schema in place; `atlas migrate diff` reports no drift.
-- add column "payer_id" to table: "claims"
ALTER TABLE `claims` ADD COLUMN `payer_id` text NULL DEFAULT '';
-- add column "npi" to table: "practice_config"
ALTER TABLE `practice_config` ADD COLUMN `npi` text NULL DEFAULT '';
-- drop column "insurance_subscriber_id" from table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_id`;
-- add column "insurance_payer_id" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_payer_id` text NULL DEFAULT '';
-- add column "insurance_subscriber_first_name" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_first_name` text NULL DEFAULT '';
-- add column "insurance_subscriber_last_name" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_last_name` text NULL DEFAULT '';
-- add column "insurance_subscriber_dob" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_dob` text NULL DEFAULT '';
-- add column "insurance_subscriber_sex" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_sex` text NULL DEFAULT '';
-- add column "insurance_subscriber_relationship" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_relationship` text NULL DEFAULT '';
-- add column "insurance_subscriber_address_line1" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_address_line1` text NULL DEFAULT '';
-- add column "insurance_subscriber_address_line2" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_address_line2` text NULL DEFAULT '';
-- add column "insurance_subscriber_city" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_city` text NULL DEFAULT '';
-- add column "insurance_subscriber_state_province" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_state_province` text NULL DEFAULT '';
-- add column "insurance_subscriber_postal_code" to table: "patients"
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_postal_code` text NULL DEFAULT '';
-- add column "npi" to table: "providers"
ALTER TABLE `providers` ADD COLUMN `npi` text NULL DEFAULT '';
-- add column "taxonomy_code" to table: "providers"
ALTER TABLE `providers` ADD COLUMN `taxonomy_code` text NULL DEFAULT '';

-- +goose Down
ALTER TABLE `providers` DROP COLUMN `taxonomy_code`;
ALTER TABLE `providers` DROP COLUMN `npi`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_postal_code`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_state_province`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_city`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_address_line2`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_address_line1`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_relationship`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_sex`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_dob`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_last_name`;
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_first_name`;
ALTER TABLE `patients` DROP COLUMN `insurance_payer_id`;
ALTER TABLE `patients` ADD COLUMN `insurance_subscriber_id` text NULL DEFAULT '';
ALTER TABLE `practice_config` DROP COLUMN `npi`;
ALTER TABLE `claims` DROP COLUMN `payer_id`;
