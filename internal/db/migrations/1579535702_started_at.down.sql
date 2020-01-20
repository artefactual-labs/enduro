ALTER TABLE collection DROP COLUMN `started_at`;

DROP INDEX `collection_started_at_idx` ON `collection`;
