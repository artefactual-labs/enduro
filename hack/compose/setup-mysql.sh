#!/usr/bin/env sh

set -eu

echo 'Starting MySQL schema setup...'
echo 'Waiting for MySQL port to be available...'
nc -z -w 10 mysql 3306
echo 'MySQL port is available'

# Set up temporal database schema.
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal setup-schema -v 0.0
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal update-schema --schema-name mysql/v8/temporal

# Set up visibility database schema.
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal_visibility setup-schema -v 0.0
temporal-sql-tool --plugin mysql8 --ep mysql -u enduro -p 3306 -pw enduro123 --db temporal_visibility update-schema --schema-name mysql/v8/visibility
