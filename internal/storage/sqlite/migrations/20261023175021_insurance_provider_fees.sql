-- +goose Up
-- create "insurance_providers" table
CREATE TABLE `insurance_providers` (
  `id` text NOT NULL,
  `name` text NOT NULL,
  `normalized_name` text NOT NULL,
  `archived` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  CHECK (length(normalized_name) > 0),
  CHECK (archived IN (0, 1))
);
-- create index "insurance_providers_normalized_name" to table: "insurance_providers"
CREATE UNIQUE INDEX `insurance_providers_normalized_name` ON `insurance_providers` (`normalized_name`);
-- create "insurance_fee_overrides" table
CREATE TABLE `insurance_fee_overrides` (
  `provider_id` text NOT NULL,
  `country_code` text NOT NULL,
  `code` text NOT NULL,
  `fee` integer NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`provider_id`, `country_code`, `code`),
  CONSTRAINT `0` FOREIGN KEY (`country_code`, `code`) REFERENCES `procedure_codes` (`country_code`, `code`) ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT `1` FOREIGN KEY (`provider_id`) REFERENCES `insurance_providers` (`id`) ON UPDATE NO ACTION ON DELETE RESTRICT,
  CHECK (fee >= 0 AND fee <= 9007199254740991)
);

-- +goose Down
-- reverse: create "insurance_fee_overrides" table
DROP TABLE `insurance_fee_overrides`;
-- reverse: create index "insurance_providers_normalized_name" to table: "insurance_providers"
DROP INDEX `insurance_providers_normalized_name`;
-- reverse: create "insurance_providers" table
DROP TABLE `insurance_providers`;
