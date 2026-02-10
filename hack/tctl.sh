#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

TEMPORAL_ADDRESS="${TEMPORAL_ADDRESS:-temporal:7233}"

docker compose run --rm --no-deps --entrypoint temporal temporal-admin-tools "$@" --address "${TEMPORAL_ADDRESS}"
