#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset
set -o xtrace

__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

go run ${__dir}/main.go > ${__dir}/data.csv

mysql -h127.0.0.1 -uroot -proot123 -P7450 enduro \
	-e "DELETE FROM collection;"

mysql -h127.0.0.1 -uroot -proot123 -P7450 enduro \
	-e "LOAD DATA LOCAL INFILE '${__dir}/data.csv' INTO TABLE collection FIELDS TERMINATED BY ','"
