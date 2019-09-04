CREATE TABLE collection (
  `id` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `name` VARCHAR(2048) NOT NULL,
  `workflow_id` VARCHAR(255) NOT NULL,
  `run_id` VARCHAR(36) NOT NULL,
  `transfer_id` VARCHAR(36) NOT NULL,
  `aip_id` VARCHAR(36) NOT NULL,
  `original_id` VARCHAR(255) NOT NULL,
  `status` TINYINT NOT NULL, -- {new, in progress, done, error, unknown}
  `created_at` TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6) NOT NULL,
  `completed_at` TIMESTAMP(6) NULL,
  PRIMARY KEY (`id`)
);
