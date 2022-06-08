CREATE TABLE collection (
  `id` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `name` VARCHAR(2048) NOT NULL,
  `workflow_id` VARCHAR(255) NOT NULL,
  `run_id` VARCHAR(36) NOT NULL,
  `aip_id` VARCHAR(36) NOT NULL,
  `status` TINYINT NOT NULL, -- {new, in progress, done, error, unknown}
  `created_at` TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6) NOT NULL,
  `started_at` TIMESTAMP(6) NULL,
  `completed_at` TIMESTAMP(6) NULL,
  PRIMARY KEY (`id`),
  KEY `collection_name_idx` (`name`(50)),
  KEY `collection_aip_id_idx` (`aip_id`),
  KEY `collection_status_idx` (`status`),
  KEY `collection_created_at_idx` (`created_at`),
  KEY `collection_started_at_idx` (`started_at`)
);
CREATE TABLE preservation_action (
  `id` INT UNSIGNED AUTO_INCREMENT NOT NULL,
  `action_id` VARCHAR(36) NOT NULL,
  `name` VARCHAR(2048) NOT NULL,
  `status` TINYINT NOT NULL, -- {unspecified, complete, processing, failed}
  `started_at` TIMESTAMP(6) NULL,
  `collection_id` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  FOREIGN KEY (`collection_id`) REFERENCES collection(`id`)
);
