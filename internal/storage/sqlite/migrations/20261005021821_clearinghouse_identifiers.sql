-- +goose Up
-- add column "payer_id" to table: "claims"
ALTER TABLE `claims` ADD COLUMN `payer_id` text NULL DEFAULT '';
-- add column "patient_control_number" to table: "claims"
ALTER TABLE `claims` ADD COLUMN `patient_control_number` text NULL DEFAULT '';
-- add column "external_claim_id" to table: "claims"
ALTER TABLE `claims` ADD COLUMN `external_claim_id` text NULL DEFAULT '';
-- add column "npi" to table: "practice_config"
ALTER TABLE `practice_config` ADD COLUMN `npi` text NULL DEFAULT '';
-- add column "taxonomy_code" to table: "practice_config"
ALTER TABLE `practice_config` ADD COLUMN `taxonomy_code` text NULL DEFAULT '';
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
-- add column "npi" to table: "providers"
ALTER TABLE `providers` ADD COLUMN `npi` text NULL DEFAULT '';
-- add column "taxonomy_code" to table: "providers"
ALTER TABLE `providers` ADD COLUMN `taxonomy_code` text NULL DEFAULT '';

-- +goose Down
-- reverse: add column "taxonomy_code" to table: "providers"
ALTER TABLE `providers` DROP COLUMN `taxonomy_code`;
-- reverse: add column "npi" to table: "providers"
ALTER TABLE `providers` DROP COLUMN `npi`;
-- reverse: add column "insurance_subscriber_relationship" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_relationship`;
-- reverse: add column "insurance_subscriber_sex" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_sex`;
-- reverse: add column "insurance_subscriber_dob" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_dob`;
-- reverse: add column "insurance_subscriber_last_name" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_last_name`;
-- reverse: add column "insurance_subscriber_first_name" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_subscriber_first_name`;
-- reverse: add column "insurance_payer_id" to table: "patients"
ALTER TABLE `patients` DROP COLUMN `insurance_payer_id`;
-- reverse: add column "taxonomy_code" to table: "practice_config"
ALTER TABLE `practice_config` DROP COLUMN `taxonomy_code`;
-- reverse: add column "npi" to table: "practice_config"
ALTER TABLE `practice_config` DROP COLUMN `npi`;
-- reverse: add column "external_claim_id" to table: "claims"
ALTER TABLE `claims` DROP COLUMN `external_claim_id`;
-- reverse: add column "patient_control_number" to table: "claims"
ALTER TABLE `claims` DROP COLUMN `patient_control_number`;
-- reverse: add column "payer_id" to table: "claims"
ALTER TABLE `claims` DROP COLUMN `payer_id`;
