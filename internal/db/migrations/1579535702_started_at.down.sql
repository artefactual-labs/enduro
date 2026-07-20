DROP INDEX `collection_started_at_idx` ON `collection`;

ALTER TABLE collection DROP COLUMN `started_at`;
