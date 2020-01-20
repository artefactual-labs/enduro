ALTER TABLE collection ADD `started_at` TIMESTAMP(6) NULL AFTER `created_at`;

CREATE INDEX `collection_started_at_idx` ON `collection` (`started_at`);
