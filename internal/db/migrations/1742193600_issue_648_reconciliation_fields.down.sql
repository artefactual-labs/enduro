ALTER TABLE `collection`
  DROP COLUMN `reconciliation_error`,
  DROP COLUMN `reconciliation_checked_at`,
  DROP COLUMN `reconciliation_status`,
  DROP COLUMN `aip_stored_at`;
