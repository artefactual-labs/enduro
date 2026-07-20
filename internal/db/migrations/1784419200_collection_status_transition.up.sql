CREATE TABLE `collection_status_transition` (
  `id` BIGINT UNSIGNED AUTO_INCREMENT NOT NULL,
  `collection_id` INT UNSIGNED NOT NULL,
  `workflow_id` VARCHAR(255) NOT NULL,
  `run_id` VARCHAR(36) NOT NULL,
  `previous_status` TINYINT NULL,
  `status` TINYINT NOT NULL,
  `occurred_at` TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6) NOT NULL,
  `is_run_start` BOOLEAN DEFAULT FALSE NOT NULL,
  `reason` VARCHAR(64) NULL,
  PRIMARY KEY (`id`),
  INDEX `collection_status_transition_collection_time_idx` (`collection_id`, `occurred_at`, `id`),
  CONSTRAINT `collection_status_transition_collection_fk`
    FOREIGN KEY (`collection_id`) REFERENCES `collection` (`id`) ON DELETE CASCADE
);
