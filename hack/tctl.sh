#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

docker compose exec temporal-admin-tools tctl $@
