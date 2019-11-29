CREATE INDEX `collection_name_idx` ON `collection` (`name`(50));
CREATE INDEX `collection_transfer_id_idx` ON `collection` (`transfer_id`(36));
CREATE INDEX `collection_aip_id_idx` ON `collection` (`aip_id`(36));
CREATE INDEX `collection_original_id_idx` ON `collection` (`original_id`(36));
CREATE INDEX `collection_pipeline_id_idx` ON `collection` (`pipeline_id`(36));
CREATE INDEX `collection_status_idx` ON `collection` (`status`);
CREATE INDEX `collection_created_at_idx` ON `collection` (`created_at`);
