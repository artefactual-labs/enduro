#!/usr/bin/env bash

set -e
set -x

SCHEMA_DIR="/etc/cadence/schema/mysql/v57/cadence/versioned"
VISIBILITY_SCHEMA_DIR="/etc/cadence/schema/mysql/v57/visibility/versioned"
DBNAME="cadence"
VISIBILITY_DBNAME="cadence_visibility"
MYSQL_ADDR="mysql"
MYSQL_USER="root"
MYSQL_PWD="root123"

cadence-sql-tool --ep $MYSQL_ADDR -u $MYSQL_USER --pw $MYSQL_PWD --db $DBNAME setup-schema -v 0.0
cadence-sql-tool --ep $MYSQL_ADDR -u $MYSQL_USER --pw $MYSQL_PWD --db $DBNAME update-schema -d $SCHEMA_DIR
cadence-sql-tool --ep $MYSQL_ADDR -u $MYSQL_USER --pw $MYSQL_PWD --db $VISIBILITY_DBNAME setup-schema -v 0.0
cadence-sql-tool --ep $MYSQL_ADDR -u $MYSQL_USER --pw $MYSQL_PWD --db $VISIBILITY_DBNAME update-schema -d $VISIBILITY_SCHEMA_DIR
