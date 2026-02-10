CREATE DATABASE IF NOT EXISTS enduro;
CREATE DATABASE IF NOT EXISTS temporal;
CREATE DATABASE IF NOT EXISTS temporal_visibility;
GRANT ALL PRIVILEGES ON enduro.* TO 'enduro'@'%';
GRANT ALL PRIVILEGES ON temporal.* TO 'enduro'@'%';
GRANT ALL PRIVILEGES ON temporal_visibility.* TO 'enduro'@'%';
