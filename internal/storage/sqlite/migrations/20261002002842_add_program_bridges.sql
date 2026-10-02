-- +goose Up
-- create "program_bridges" table
CREATE TABLE `program_bridges` (`name` text NOT NULL, `enabled` integer NOT NULL DEFAULT 0, `path` text NOT NULL DEFAULT '', `args` text NOT NULL DEFAULT '', `created_at` datetime NOT NULL, `updated_at` datetime NOT NULL, PRIMARY KEY (`name`));

-- +goose Down
-- reverse: create "program_bridges" table
DROP TABLE `program_bridges`;
