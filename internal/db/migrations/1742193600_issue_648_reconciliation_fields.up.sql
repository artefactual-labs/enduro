ALTER TABLE `collection`
  ADD COLUMN `aip_stored_at` DATETIME(6) NULL AFTER `completed_at`,
  ADD COLUMN `reconciliation_status` VARCHAR(32) NULL AFTER `aip_stored_at`,
  ADD COLUMN `reconciliation_checked_at` DATETIME(6) NULL AFTER `reconciliation_status`,
  ADD COLUMN `reconciliation_error` TEXT NULL AFTER `reconciliation_checked_at`;
